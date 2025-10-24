package music

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

func Skip(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		return
	}
	channelID, status := getUserVoiceChannel(s, i)
	if status == Unknown {
		message := fmt.Sprintf("<@%s> You are not in a voice channel", i.Interaction.Member.User.ID)
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &message,
		})
		if err != nil {
			log.Printf("Error sending interaction response: %v\n", err)
		}
		return
	}
	if vc, exists := voiceConnections[i.GuildID]; exists && vc.ChannelID != channelID {
		message := fmt.Sprintf("<@%s> Why are you trying to skip music when you aren't even listening to it? stupid dumb idiot", i.Interaction.Member.User.ID)
		type ModifyGuildMemberParams struct {
			Nick                       string   `json:"nick,omitempty"`
			Roles                      []string `json:"roles,omitempty"`
			Mute                       bool     `json:"mute,omitempty"`
			Deaf                       bool     `json:"deaf,omitempty"`
			ChannelID                  string   `json:"channel_id,omitempty"`
			CommunicationDisabledUntil string   `json:"communication_disabled_until,omitempty"`
			Flags                      int      `json:"flags,omitempty"`
		}
		params := ModifyGuildMemberParams{
			ChannelID: vc.ChannelID,
		}
		jsonParams, err := json.Marshal(params)
		if err != nil {
			log.Printf("Error marshalling json: %v\n", err)
			return
		}
		jsonData := bytes.NewBuffer(jsonParams)
		req, err := http.NewRequest("PATCH", fmt.Sprintf("https://discord.com/api/v9/guilds/%s/members/%s", i.GuildID, i.Interaction.Member.User.ID), jsonData)
		if err != nil {
			log.Printf("Error moving user: %v\n", err)
		}
		err = godotenv.Load(".env")

		if err != nil {
			log.Println("Error loading .env file in skip command to move user")
		}
		token := os.Getenv("discord_token")
		req.Header.Set("Authorization", fmt.Sprintf("Bot %s", token))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal("Error sending request:", err)
		}
		defer resp.Body.Close()
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &message,
		})
		if err != nil {
			log.Printf("Error sending checking message: %v\n", err)
			return
		}
		return
	}
	if queues[guildID] == nil {
		message := fmt.Sprintf("<@%s> There is no song to skip", i.Interaction.Member.User.ID)
		_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &message,
		})
		if err != nil {
			log.Printf("Error sending interaction response: %v\n", err)
		}
		return
	}
	message := fmt.Sprintf("<@%s> has skipped the current song", i.Interaction.Member.User.ID)
	defer func() {
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &message,
		})
		if err != nil {
			log.Printf("Error sending interaction response: %v\n", err)
		}
	}()
	queuesMutex.Lock()
	defer queuesMutex.Unlock()

	if signal, ok := skipSignal[guildID]; ok {
		select {
		case signal <- true:
		default:
		}
	}
}
