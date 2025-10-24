package commands

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bwmarrin/discordgo"
)

func RandomCat(s *discordgo.Session, m *discordgo.InteractionCreate) {
	resp, err := http.Get("https://api.thecatapi.com/v1/images/search")
	if err != nil {
		fmt.Println("Error getting random fox:", err)
		return
	}
	defer resp.Body.Close()

	var result []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Println("Error decoding JSON response:", err)
		return
	}

	imageURL, ok := result[0]["url"].(string)
	if !ok {
		fmt.Println("Error extracting image URL from response")
		return
	}

	err = s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: imageURL,
		},
	})
	if err != nil {
		fmt.Println("Error sending interaction response:", err)
	}
}
