package commands

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

func RandomFact(s *discordgo.Session, m *discordgo.InteractionCreate) {
	resp, err := http.Get("https://uselessfacts.jsph.pl/api/v2/facts/random")
	if err != nil {
		fmt.Println("Error getting random fox:", err)
		return
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Println("Error decoding JSON response:", err)
		return
	}

	randomFact, ok := result["text"].(string)
	if !ok {
		fmt.Println("Error extracting image URL from response")
		return
	}

	err = s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: randomFact,
		},
	})
	if err != nil {
		fmt.Println("Error sending interaction response:", err)
	}
}
