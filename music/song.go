package music

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

func SongData(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string, index int64) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		return
	}
	if len(queues[i.GuildID]) == 0 {
		message := fmt.Sprintf("<@%s> There are no songs in the queue", i.Interaction.Member.User.ID)
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &message,
		})
		if err != nil {
			log.Printf("Error sending interaction response: %v\n", err)
		}
		return
	}
	index = index - 1
	if (index < 0) || (index >= int64(len(queues[guildID]))) {
		message := "Index out of range"
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &message,
		})
		if err != nil {
			fmt.Println("Error sending interaction response:", err)
		}
		return
	}
	message := fmt.Sprintf("<@%s> Title: %s\nAuthor: %s\nDuration: %s", i.Interaction.Member.User.ID, queues[guildID][index].Title, queues[guildID][index].Author, queues[guildID][index].Length)
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &message,
	})
	if err != nil {
		log.Printf("Error sending interaction response: %v\n", err)
	}
}
