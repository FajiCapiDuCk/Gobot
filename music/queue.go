package music

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

func Queue(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		return
	}

	if len(queues[guildID]) == 0 {
		message := fmt.Sprintf("<@%s> There are no songs in the queue", i.Interaction.Member.User.ID)
		_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &message,
		})
		if err != nil {
			log.Printf("Error sending interaction response: %v\n", err)
		}
		return
	} else if currentsong[guildID] == (Song{}) {
		message := fmt.Sprintf("<@%s> There is no song currently playing", i.Interaction.Member.User.ID)
		_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &message,
		})
		if err != nil {
			log.Printf("Error sending interaction response: %v\n", err)
		}
		return
	}

	guild, err := s.State.Guild(i.GuildID)
	if err != nil {
		log.Printf("Error fetching guild info: %v", err)
		return
	}

	var thumbnailURL string
	if guild.Icon != "" {
		thumbnailURL = fmt.Sprintf("https://cdn.discordapp.com/icons/%s/%s.png", guild.ID, guild.Icon)
	} else {
		thumbnailURL = "https://cdn.discordapp.com/attachments/654963290744684554/1326989982757556254/image.png?ex=67816f3a&is=67801dba&hm=e221e520fa544cd0f3a965915fe92e34c875b85248b366b3123412e0f698a167&"
	}

	var fields []*discordgo.MessageEmbedField

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:  fmt.Sprintf("Current song: %s\n(requested by: %s)", currentsong[guildID].Title, currentsong[guildID].Requester),
		Value: fmt.Sprintf("Duration: %s\nAuthor: %s", currentsong[guildID].Length, currentsong[guildID].Author),
	})
	for i, song := range queues[guildID] {
		field := &discordgo.MessageEmbedField{
			Name:  fmt.Sprintf("%d - %s\n(requested by: %s)", i+1, song.Title, song.Requester),
			Value: fmt.Sprintf("Duration: %s\nAuthor: %s", song.Length, song.Author),
		}
		if i == 9 {
			field.Value += fmt.Sprintf("\n**And %d more...**", len(queues[guildID])-i)
			fields = append(fields, field)
			break
		}
		fields = append(fields, field)
	}

	embed := &discordgo.MessageEmbed{
		Title:       guild.Name + " - Queue",
		Description: "Here’s the current queue:",
		Fields:      fields,
		Color:       0x00ffcc,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: thumbnailURL,
		},
	}

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
	if err != nil {
		log.Printf("Error sending interaction response: %v\n", err)
	}
}
