package music

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

func NowPlaying(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if currentsong[i.GuildID] == (Song{}) && len(queues[i.GuildID]) == 0 {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No song is currently playing",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Printf("Error sending checking message: %v\n", err)
			return
		}
		return
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Error sending interaction response: %v\n", err)
		return
	}
	message := "there you go"
	defer func() {
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &message,
		})
		if err != nil {
			log.Printf("Error editing interaction response: %v\n", err)
		}
	}()

	CurrentlyPlaying(s, i.ChannelID, currentsong[i.GuildID])
}
