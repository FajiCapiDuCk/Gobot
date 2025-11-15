package commands

import (
	"fmt"
	"log"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func RandomNumber(s *discordgo.Session, i *discordgo.InteractionCreate, count int, max uint32, dnd bool) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "<a:pislicesd20:1416741138269999145>",
		},
	})
	r := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano())))
	values := make([]uint32, count)
	for i := range count {
		values[i] = r.Uint32N(max + 1)
		if dnd {
			values[i]++
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
	r := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano())))
	d10 := r.Uint32N(11)
	d10++
	message := fmt.Sprintf("Congrats you got %d", d10)
	time.Sleep(time.Second * 2)

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &message,
	})
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
	r := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano())))
	d20 := r.Uint32N(21)
	d20++
	message := fmt.Sprintf("Congrats you got %d", d20)
	time.Sleep(time.Second * 2)

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &message,
	})
}
