package music

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func Remove(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string, index int64) {
	queuesMutex.Lock()
	defer queuesMutex.Unlock()
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if len(queues[guildID]) == 0 {
		message := "Queue is empty"
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &message,
		})
		if err != nil {
			fmt.Println("Error sending interaction response:", err)
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
	title := queues[guildID][index].Title
	queues[guildID] = append(queues[guildID][:index], queues[guildID][index+1:]...)
	if err != nil {
		fmt.Println("Error acknowledging interaction:", err)
	}
	message := fmt.Sprintf("<@%s> has removed %s from the queue", i.Member.User.ID, title)
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &message,
	})
	if err != nil {
		fmt.Println("Error sending interaction response:", err)
	}
}
