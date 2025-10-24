package music

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

func Stop(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		return
	}
	channelID, _ := getUserVoiceChannel(s, i)
	if vc, exists := voiceConnections[i.GuildID]; exists && vc.ChannelID != channelID {
		message := fmt.Sprintf("<@%s> Stopping music when you aren't even in the same voice channel as me?", i.Interaction.Member.User.ID)
		_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &message,
		})
		if err != nil {
			log.Printf("Error sending checking message: %v\n", err)
			return
		}
		return
	}
	queuesMutex.Lock()
	defer queuesMutex.Unlock()
	queues[guildID] = nil
	delete(queues, guildID)
	if signal, ok := skipSignal[guildID]; ok {
		select {
		case signal <- true:
		default:
		}
	}
	message := fmt.Sprintf("<@%s> has stopped the music", i.Interaction.Member.User.ID)
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &message,
	})
	if err != nil {
		log.Printf("Error sending interaction response: %v\n", err)
	}
}
