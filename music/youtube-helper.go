/*
 * TODO: add search query from youtube itself because why not
 */
package music

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func playYouTubeAudio(s *discordgo.Session, i *discordgo.InteractionCreate, url string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("Error sending thinking message: %v\n", err)
		return
	}
	message := fmt.Sprintf("<@%s> Queue limit has been reached(Max allowed size is %d)", i.Member.User.ID, MaxQueueSize)

	if len(queues[i.GuildID]) > MaxQueueSize {
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &message,
		})
		if err != nil {
			log.Printf("Error sending checking message: %v\n", err)
			return
		}
		return
	}
	if isPlaylistURL(url) {
		err := processPlaylist(s, i, url)
		if err == nil || err == errors.New("Queue limit has been reached") {
			return
		}
	}
	// err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
	// 	Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	// })
	// if err != nil {
	// 	fmt.Println("error sending thinking message")
	// }
	type VideoMetadata struct {
		Title     string  `json:"title"`
		Author    string  `json:"uploader"`
		Duration  float64 `json:"duration"`
		Views     int64   `json:"view_count"`
		Thumbnail string  `json:"thumbnail"`
	}

	cmd := exec.Command("yt-dlp", "-J", url)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error retrieving video: %v\n", err)
		return
	}

	var video VideoMetadata
	if err := json.Unmarshal(output, &video); err != nil {
		log.Printf("Error parsing JSON for the video: %v\n", err)
		return
	}

	song := Song{
		Title:     video.Title,
		Length:    fmt.Sprintf("%d:%02d", int(video.Duration)/60, int(video.Duration)%60),
		Author:    video.Author,
		URL:       url,
		Thumbnail: video.Thumbnail,
		Requester: i.Member.User.Username,
	}

	embeds := []*discordgo.MessageEmbed{
		{
			Title:       fmt.Sprintf("Song added: %s", song.Title),
			URL:         song.URL,
			Description: fmt.Sprintf("Song added by %s", i.Member.User.Username),
			Color:       0xFF0000,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Duration ⏳",
					Value:  song.Length,
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
				URL: song.Thumbnail,
			},
		},
	}

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &embeds,
	})
	if err != nil {
		log.Printf("Error sending interaction response: %v\n", err)
	}

	queuesMutex.Lock()
	queues[i.GuildID] = append(queues[i.GuildID], song)
	queuesMutex.Unlock()

	go PlayNextInQueue(s, i.GuildID, i)
}

func isPlaylistURL(url string) bool {
	return strings.Contains(url, "list=")
}

func isYouTubeURL(url string) bool {
	return strings.HasPrefix(url, "https://www.youtube.com") || strings.HasPrefix(url, "https://music.youtube.com") || strings.HasPrefix(url, "https://youtu.be")
}

func processPlaylist(s *discordgo.Session, i *discordgo.InteractionCreate, url string) error {
	if len(queues[i.GuildID]) > MaxQueueSize {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("<@%s> Queue limit has been reached (Max allowed size is %d)", i.Member.User.ID, MaxQueueSize),
			},
		})
		if err != nil {
			log.Printf("Error sending checking message: %v\n", err)
			return err
		}
		return errors.New("Queue limit has been reached")
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	// For some reason was sending HTTP: 400 error code cant be asked to figure out why
	// if err != nil {
	// 	log.Printf("Error acknowledging interaction: %v", err)
	// 	return err
	// }

	cmd := exec.Command("yt-dlp", "-J", "--flat-playlist", "--playlist-end", "100", url)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error fetching playlist: %v\n", err)
		_, _ = s.ChannelMessageSend(i.ChannelID, "Error fetching playlist. Possible reasons: it's a private playlist or unknown error occured.")
		return err
	}

	type Video struct {
		Title      string  `json:"title"`
		Duration   float64 `json:"duration"`
		Uploader   string  `json:"uploader"`
		ID         string  `json:"id"`
		Thumbnails []struct {
			URL string `json:"url"`
		} `json:"thumbnails"`
	}

	type Playlist struct {
		Entries []Video `json:"entries"`
		Title   string  `json:"title"`
	}

	var playlist Playlist
	if err := json.Unmarshal(output, &playlist); err != nil {
		log.Printf("Error parsing playlist JSON: %v\n", err)
		_, _ = s.ChannelMessageSend(i.ChannelID, "Error parsing playlist data.")
		return err
	}
	var number int
	for index := range playlist.Entries {
		video := &playlist.Entries[index]
		if index > 101 {
			break
		}
		song := Song{
			Title: video.Title,
			Length: func(seconds float64) string {
				if seconds < 0 {
					return "Unknown"
				}
				hours := int(seconds) / 3600
				minutes := (int(seconds) % 3600) / 60
				secs := int(seconds) % 60

				if hours > 0 {
					return fmt.Sprintf("%dh%02dm%02ds", hours, minutes, secs)
				}
				return fmt.Sprintf("%02dm%02ds", minutes, secs)
			}(video.Duration),
			Author:    video.Uploader,
			URL:       fmt.Sprintf("https://www.youtube.com/watch?v=%s", video.ID),
			Thumbnail: video.Thumbnails[len(video.Thumbnails)-1].URL,
			Requester: i.Member.User.Username,
		}
		number++
		queuesMutex.Lock()
		if len(queues[i.GuildID]) > MaxQueueSize {
			message := fmt.Sprintf("<@%s> Queue limit has been reached (Max allowed size is %d)", i.Member.User.ID, MaxQueueSize)
			_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &message,
			})
			if err != nil {
				log.Printf("Error sending checking message: %v\n", err)
				queuesMutex.Unlock()
				return err
			}
			queuesMutex.Unlock()
			return errors.New("Queue limit has been reached")
		}
		queues[i.GuildID] = append(queues[i.GuildID], song)
		queuesMutex.Unlock()
	}
	if number == MaxQueueSize {
		_, err = s.ChannelMessageSend(i.ChannelID, "Playlist has more than 100 videos. The playlist has been truncated.(To Stop from adding thousands big playlist)")
	}
	if err != nil {
		log.Printf("Error sending playlist response: %v\n", err)
	}
	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Title:       fmt.Sprintf("%s added playlist: %s", i.Member.User.Username, playlist.Title),
				Description: fmt.Sprintf("%d videos added to the queue", number),
				Color:       0x1DA1F2,
			},
		},
	})
	if err != nil {
		log.Printf("Error sending playlist response: %v\n", err)
	}

	queuesMutex.Lock()
	if len(queues[i.GuildID]) == number {
		go PlayNextInQueue(s, i.GuildID, i)
	}
	queuesMutex.Unlock()
	return nil
}
