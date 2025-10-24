package commands

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

func Help(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
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
	embed := &discordgo.MessageEmbed{
		Title: "Available commands",
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "/play",
				Value: "Play a song from YouTube or SoundCloud",
			},
			{
				Name:  "/playnext",
				Value: "Play a song from YouTube or SoundCloud and add it to start of the queue",
			},
			{
				Name:  "/queue",
				Value: "Show the current queue",
			},
			{
				Name:  "/skip",
				Value: "Skip the current song",
			},
			{
				Name:  "/stop",
				Value: "Stops the current song and clears the queue",
			},
			{
				Name:  "/song",
				Value: "Show data about a song from the queue",
			},
			{
				Name:  "/remove",
				Value: "Remove a song from the queue",
			},
			{
				Name:  "/nowplaying",
				Value: "Show the current song",
			},
			{
				Name:  "/save",
				Value: "Save the current song to your DMs",
			},
			{
				Name:  "/cat",
				Value: "Send a random cat picture",
			},
			{
				Name:  "/dog",
				Value: "Send a random dog picture",
			},
			{
				Name:  "/fox",
				Value: "Send a random fox picture",
			},
			{
				Name:  "/prikol",
				Value: "Send a random fact in chat",
			},
		},
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
