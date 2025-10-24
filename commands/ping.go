package commands

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

func PingCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	start := time.Now().UnixMilli()
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Pinging penguins",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})

	if err != nil {
		fmt.Println("Error sending interaction response:", err)
		return
	}

	latency := time.Now().UnixMilli() - start

	message := fmt.Sprintf("Pong! Latency is %dms 🚀 ", latency)

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &message,
	})

	if err != nil {
		fmt.Println("Error editing interaction response:", err)
	}
}
