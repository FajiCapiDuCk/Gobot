package commands

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func RandomNumber(s *discordgo.Session, i *discordgo.InteractionCreate, count int, max int64, dnd bool) {
	if max <= 0 {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Max cannot be 0 or a negative number",
		},
	})	
		return
	}
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "<a:pislicesd20:1416741138269999145>",
		},
	})
	values := make([]int64, count)
	for i := range count {
		values[i] = randomNumbergen(max)
		if !dnd {
			values[i]--
		}
	}
	var builder strings.Builder
	builder.WriteString("Results are: \n")
	for _, value := range values {
		builder.WriteString(strconv.Itoa(int(value)))
		builder.WriteString(" ")
	}

	result := builder.String()
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &result,
	})

	if err != nil {
		log.Printf("ERROR: Unable to send message for throwing general dice\n%s", err)
	}
}

func D10(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "<a:pislicesd20:1416741138269999145>",
		},
	})
	if err != nil {
		log.Printf("ERROR: Unable to send message for D20 throw\n%s", err)
	}
    
    	message := fmt.Sprintf("Congrats you got %d", randomNumbergen(10))
	time.Sleep(time.Second * 2)

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &message,
	})
}
func randomNumbergen(n int64) int64 {
	d10, _ := rand.Int(rand.Reader, big.NewInt(10))
	return int64(d10.Int64()) + 1
}

func D20(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "<a:pislicesd20:1416741138269999145>",
		},
	})
	if err != nil {
		log.Printf("ERROR: Unable to send message for D10 throw\n%s", err)
	}
	message := fmt.Sprintf("Congrats you got %d", randomNumbergen(20))
	time.Sleep(time.Second * 2)

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &message,
	})
}
