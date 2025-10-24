package music

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"

	"github.com/bwmarrin/discordgo"
)

func Save(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if currentsong[i.GuildID] == (Song{}) {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "There is no song currently playing",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Printf("Error acknowledging interaction: %v", err)
			return
		}
		return
	}
	channel, err := s.UserChannelCreate(i.Member.User.ID)
	if err != nil {
		log.Printf("Error creating user channel: %v\n", err)
		s.ChannelMessageSend(
			i.ChannelID,
			"Something went wrong while sending the DM!",
		)
		return
	}
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Song saved to your DMs!",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Error acknowledging interaction: %v", err)
		return
	}
	saveSong(s, channel.ID, currentsong[i.GuildID])
}

type VideoMetadata struct {
	Title     string  `json:"title"`
	Author    string  `json:"uploader"`
	Duration  float64 `json:"duration"`
	Views     int64   `json:"view_count"`
	Thumbnail string  `json:"thumbnail"`
}

func saveSong(s *discordgo.Session, channelID string, song Song) {
	if isYouTubeURL(song.URL) {
		cmd := exec.Command("yt-dlp", "-J", song.URL)
		output, err := cmd.Output()
		if err != nil {
			log.Fatalf("Error retrieving video: %v", err)
		}

		var video VideoMetadata
		if err := json.Unmarshal(output, &video); err != nil {
			log.Fatalf("Error parsing JSON: %v", err)
		}

		embed := &discordgo.MessageEmbed{
			Title:       video.Title,
			URL:         song.URL,
			Description: fmt.Sprintf("Song by %s", video.Author),
			Color:       0x0000FF,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Duration ⏳",
					Value:  fmt.Sprintf("%d:%02d", int(video.Duration)/60, int(video.Duration)%60),
					Inline: true,
				},
				{
					Name: "Views 👀",
					Value: func(n int64) string {
						if n < 1000 {
							return fmt.Sprint(n)
						}
						if n < 1000000 {
							return fmt.Sprintf("%.2fK", float64(n)/1000)
						}
						if n < 1000000000 {
							return fmt.Sprintf("%.2fM", float64(n)/1000000)
						}
						return fmt.Sprintf("%.2fB", float64(n)/1000000000)
					}(video.Views),
					Inline: true,
				},
			},
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: video.Thumbnail,
			},
		}

		_, err = s.ChannelMessageSendEmbed(channelID, embed)
		if err != nil {
			log.Fatalf("Error sending message: %v", err)
		}
	}
}
