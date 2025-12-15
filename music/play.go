/*
	 TODO : make so it works in multiple servers at once??
		fix so it doesnt panic if user isnt in a voice channel
		add checks if bot is in a voice channel because lonely is a funny person
*/
package music

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
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

// Enum for VoiceChannelStatus
type VoiceChannelStatus int

const (
	Hidden VoiceChannelStatus = iota
	Public
	Unknown
)

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
	// Starts the processes in a group so I can murder them later
	ytDlpCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	ffmpegCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

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

func KillProcessTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	time.Sleep(100 * time.Millisecond)
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	_, _ = cmd.Process.Wait() // Reap the zombie
	return nil
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
