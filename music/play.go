/*
	 TODO : make so it works in multiple servers at once??
		fix so it doesnt panic if user isnt in a voice channel
		add search query from youtube itself because why not
		add checks if bot is in a voice channel because lonely is a funny person
*/
package music

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/dgvoice"
	"github.com/bwmarrin/discordgo"
)

type Song struct {
	Title     string
	Length    string
	URL       string
	Author    string
	Views     string
	Thumbnail string
	Requester string
}

var (
	queues      = make(map[string][]Song)
	playing     = make(map[string]bool)
	queuesMutex sync.Mutex
	skipSignal  = make(map[string]chan bool)
	currentsong = make(map[string]Song)
)

const (
	SampleRate      = 48000
	Channels        = 2
	bufferSize      = 960
	MaxQueueSize    = 200
	MaxPlayListSize = 100
)

type VoiceChannelStatus int

const (
	Hidden VoiceChannelStatus = iota
	Public
	Unknown
)

func isPlaylistURL(url string) bool {
	return strings.Contains(url, "list=")
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

func CurrentlyPlaying(session *discordgo.Session, channelID string, song Song) {
	if isYouTubeURL(song.URL) {
		type VideoMetadata struct {
			Title     string  `json:"title"`
			Author    string  `json:"uploader"`
			Duration  float64 `json:"duration"`
			Views     int64   `json:"view_count"`
			Thumbnail string  `json:"thumbnail"`
		}
		cmd := exec.Command("yt-dlp", "-J", song.URL)
		output, err := cmd.Output()
		if err != nil {
			log.Printf("Error retrieving video: %v\n", err)
		}

		var video VideoMetadata
		if err := json.Unmarshal(output, &video); err != nil {
			log.Printf("Error parsing JSON: %v\n", err)
		}

		author := video.Author
		thumbnail := video.Thumbnail
		_, err = session.ChannelMessageSend(channelID, "Now playing:")
		if err != nil {
			fmt.Println(err)
		}

		embed := &discordgo.MessageEmbed{
			Title:       video.Title,
			URL:         song.URL,
			Description: fmt.Sprintf("Song by %s", author),
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
				URL: thumbnail,
			},
		}
		buttons := []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Skip",
						Style:    discordgo.PrimaryButton,
						CustomID: "skip_button",
					},
					discordgo.Button{
						Label:    "Save",
						Style:    discordgo.SuccessButton,
						CustomID: "save_button",
					},
					discordgo.Button{
						Label:    "Queue",
						Style:    discordgo.SecondaryButton,
						CustomID: "queue_button",
					},
					discordgo.Button{
						Label: "Open",
						Style: discordgo.LinkButton,
						URL:   song.URL,
					},
				},
			},
		}

		_, err = session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Embed:      embed,
			Components: buttons,
		})
		if err != nil {
			log.Fatalf("Error sending message: %v", err)
		}
	}
}

func isYouTubeURL(url string) bool {
	return strings.HasPrefix(url, "https://www.youtube.com") || strings.HasPrefix(url, "https://music.youtube.com") || strings.HasPrefix(url, "https://youtu.be")
}
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

func PlayAudio(s *discordgo.Session, i *discordgo.InteractionCreate, url string) {
	if s == nil || i == nil {
		log.Fatal("Session or InteractionCreate is nil")
	}

	if len(queues[i.GuildID]) > MaxQueueSize {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("<@%s> Queue limit has been reached (Max allowed size is %d)", i.Member.User.ID, MaxQueueSize),
			},
		})
		if err != nil {
			log.Printf("Error sending checking message: %v\n", err)
			return
		}
		return
	}

	if !isYouTubeURL(url) {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "URL is not from YouTube",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Printf("Error sending checking message: %v\n", err)
			return
		}
		return
	}
	channelID, status := getUserVoiceChannel(s, i)
	if channelID == "" {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("<@%s> You are not in a voice channel", i.Member.User.ID),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Printf("Error sending checking message: %v\n", err)
			return
		}
		return
	}
	if status == Hidden {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("<@%s> You are in a hidden voice channel, I cannot join it :(", i.Member.User.ID),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Printf("Error sending checking message: %v\n", err)
			return
		}
		return
	}
	if vc, exists := voiceConnections[i.GuildID]; exists && vc.ChannelID != channelID {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("<@%s> Songs are already being played in another voice channel, join that voice channel <3", i.Member.User.ID),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Printf("Error sending checking message: %v\n", err)
			return
		}
		return
	}
	if isYouTubeURL(url) {
		playYouTubeAudio(s, i, url)
	}
}

func PlayNextInQueue(s *discordgo.Session, guildID string, i *discordgo.InteractionCreate) {
	queuesMutex.Lock()
	if playing[guildID] || len(queues[guildID]) == 0 {
		queuesMutex.Unlock()
		return
	}
	playing[guildID] = true
	skipSignal[guildID] = make(chan bool, 1)
	queuesMutex.Unlock()

	go func(guildID string) {
		for {
			queuesMutex.Lock()
			if len(queues[guildID]) == 0 {
				playing[guildID] = false
				queuesMutex.Unlock()
				leaveVoiceChannel(s, i, guildID)
				return
			}
			if vc, exists := voiceConnections[guildID]; exists {
				if !checkOtherMembersInVoiceChannel(s, guildID) {
					currentsong[guildID] = (Song{})
					close(skipSignal[guildID])
					delete(voiceConnections, guildID)
					delete(playing, guildID)
					delete(currentsong, guildID)
					delete(queues, guildID)
					delete(skipSignal, guildID)
					queuesMutex.Unlock()
					_ = vc.Disconnect()
					_, err := s.ChannelMessageSend(i.ChannelID, "No one is in the voice channel, leaving and clearing the queue")
					if err != nil {
						log.Printf("Error sending leaving message: %v\n", err)
					}
					return
				}
			}
			song := queues[guildID][0]
			currentsong[guildID] = song
			queues[guildID] = queues[guildID][1:]
			queuesMutex.Unlock()

			vc, err := joinUserVoiceChannel(s, i)
			if err != nil {
				log.Printf("Error joining voice channel: %v\n", err)
				continue
			}

			CurrentlyPlaying(s, i.ChannelID, song)
			err = streamAudio(vc, song.URL, guildID)
			if err != nil {
				log.Printf("Error streaming audio: %v\n", err)
				continue
			}
			currentsong[guildID] = Song{}
		}
	}(guildID)
}
func KillProcessTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}

	if runtime.GOOS != "windows" {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		time.Sleep(100 * time.Millisecond) // Give time to exit
	}
	switch runtime.GOOS {
	case "windows":
		kill := exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", cmd.Process.Pid))
		return kill.Run()
	default:
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		_, _ = cmd.Process.Wait() // Reap the zombie
		return nil
	}
}

func streamAudio(vc *discordgo.VoiceConnection, url, guildID string) error {
	defer func() {
		err := vc.Speaking(false) // Stop speaking when done
		if err != nil {
			_ = fmt.Errorf("ERROR: Couldn't stop speaking: %v", err)
		}
	}()
	// Set up yt-dlp and ffmpeg commands
	ytDlpCmd := exec.Command("yt-dlp", "-f", "bestaudio", "--extract-audio", "--audio-quality", "0", "--no-playlist", "-o", "-", url)
	ffmpegCmd := exec.Command(
		"ffmpeg",
		"-i", "-",
		"-ar", strconv.Itoa(SampleRate),
		"-ac", strconv.Itoa(Channels),
		"-f", "s16le",
		"pipe:1",
		"-loglevel", "error",
		"-nostats",
		"-vn",
	)

	// Ensure processes are killed when done (cross-platform)
	defer func() {
		if ytDlpCmd.Process != nil {
			KillProcessTree(ytDlpCmd)
		}
		if ffmpegCmd.Process != nil {
			KillProcessTree(ffmpegCmd)
		}
	}()

	// Set up pipes
	ytDlpOut, ytDlpIn := io.Pipe()
	ytDlpCmd.Stdout = ytDlpIn
	ytDlpCmd.Stderr = os.Stderr
	ffmpegCmd.Stdin = ytDlpOut

	ffmpegOut, err := ffmpegCmd.StdoutPipe()
	if err != nil {
		_ = fmt.Errorf("ERROR: Piping ffmpeg failed: %v", err)
		return err
	}

	// Start processes in a new process group (Unix)
	if runtime.GOOS != "windows" {
		ytDlpCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		ffmpegCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	err = ytDlpCmd.Start()
	if err != nil {
		_ = fmt.Errorf("ERROR: yt-dlp Start Error: %v", err)
		return err
	}
	err = ffmpegCmd.Start()
	if err != nil {
		_ = fmt.Errorf("ERROR: ffmpeg Start Error: %v", err)
		return err
	}

	// Ensure yt-dlp closes properly
	go func() {
		defer ytDlpIn.Close()
		_ = ytDlpCmd.Wait() // Avoid zombie process
	}()

	ffmpegBuf := bufio.NewReaderSize(ffmpegOut, 4096)
	send := make(chan []int16, 2)
	defer close(send)

	closeChan := make(chan bool)
	go func() {
		dgvoice.SendPCM(vc, send)
		closeChan <- true
	}()

	// Start speaking on the voice channel
	err = vc.Speaking(true)
	if err != nil {
		_ = fmt.Errorf("ERROR: Couldn't set speaking status to true: %v", err)
		return err
	}

	for {
		pcmBuffer := make([]int16, bufferSize*Channels)
		select {
		case <-skipSignal[guildID]: // Handle skip signal
			log.Println("Skipping current song...")
			return nil
		default:
		}

		// Read PCM data from ffmpeg
		err = binary.Read(ffmpegBuf, binary.LittleEndian, &pcmBuffer)
		if err == io.EOF {
			log.Println("End of stream reached")
			return nil
		}

		select {
		case send <- pcmBuffer:
		case <-closeChan:
			return nil
		}

	}
}

var voiceConnections = make(map[string]*discordgo.VoiceConnection)

func joinUserVoiceChannel(s *discordgo.Session, i *discordgo.InteractionCreate) (*discordgo.VoiceConnection, error) {
	channelID, status := getUserVoiceChannel(s, i)
	if channelID == "" {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You are not in a voice channel",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Printf("Error sending checking message: %v\n", err)
			return nil, err
		}
		return nil, fmt.Errorf("user not in a voice channel")
	}
	if status == Hidden {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You are in a hidden voice channel, I cannot join it :(",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Printf("Error sending checking message: %v\n", err)
			return nil, err
		}
		return nil, fmt.Errorf("user not in a voice channel")
	}
	if vc, exists := voiceConnections[i.GuildID]; exists && vc.ChannelID == channelID && vc.Ready {
		return vc, nil
	}

	vc, err := s.ChannelVoiceJoin(i.GuildID, channelID, false, true)
	if err == nil {
		voiceConnections[i.GuildID] = vc
	} else {
		log.Printf("Error joining voice channel: %v\n", err)
	}
	return vc, err
}

func checkOtherMembersInVoiceChannel(s *discordgo.Session, guildID string) bool {
	vc := voiceConnections[guildID]

	guild, err := s.State.Guild(guildID)
	if err != nil {
		log.Printf("Error getting guild state: %v\n", err)
		return false
	}

	for _, vs := range guild.VoiceStates {
		if vs.ChannelID == vc.ChannelID && vs.UserID != s.State.User.ID {
			return true
		}
	}
	return false
}

func leaveVoiceChannel(s *discordgo.Session, i *discordgo.InteractionCreate, guildID string) {
	if vc, exists := voiceConnections[guildID]; exists {
		_ = vc.Disconnect()
		currentsong[guildID] = (Song{})
		delete(voiceConnections, guildID)
		delete(playing, guildID)
		delete(currentsong, guildID)
		delete(queues, guildID)
		close(skipSignal[guildID])
		delete(skipSignal, guildID)
		_, err := s.ChannelMessageSend(i.ChannelID, "Queue empty, leaving voice channel")
		if err != nil {
			log.Printf("Error sending leaving message: %v\n", err)
		}
	} else {
		log.Printf("No voice connection to disconnect for guild %s\n", guildID)
	}
}

func getUserVoiceChannel(s *discordgo.Session, i *discordgo.InteractionCreate) (string, VoiceChannelStatus) {
	guild, err := s.State.Guild(i.GuildID)
	if err != nil {
		return "", Unknown
	}
	for _, vs := range guild.VoiceStates {
		if vs.UserID == i.Member.User.ID {
			channel, err := s.State.Channel(vs.ChannelID)
			if err != nil {
				channel, err = s.Channel(vs.ChannelID)
				if err != nil {
					return "", Unknown
				}
			}
			perms, err := s.UserChannelPermissions(s.State.User.ID, channel.ID)
			if err != nil {
				return "", Unknown
			}
			if perms&discordgo.PermissionViewChannel == 0 {
				return "", Hidden
			} else if perms&discordgo.PermissionViewChannel != 0 {
				return vs.ChannelID, Public
			}

			return "", Unknown
		}
	}
	return "", Unknown
}
