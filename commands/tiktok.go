package commands

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func isTiktokDomain(u string) bool {
	pattern := `^(https?:\/\/)?([a-zA-Z0-9-]+\.)*tiktok\.com\/?`
	re := regexp.MustCompile(pattern)
	return re.MatchString(u)
}

func Processtiktok(s *discordgo.Session, tiktokURL string, m *discordgo.InteractionCreate) error {
	if !isTiktokDomain(tiktokURL) {
		err := s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Domain isn't owned by TikTok",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Printf("Error acknowledging interaction: %v\n", err)
			return fmt.Errorf("error acknowledging interaction: %w", err)
		}
		return nil
	}

	err := s.InteractionRespond(m.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Checking...",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Error sending checking message: %v\n", err)
		return fmt.Errorf("error acknowledging interaction: %w", err)
	}

	resp, err := http.Get(tiktokURL)
	if err != nil {
		log.Printf("Failed to get URL: %v\n", err)
		return fmt.Errorf("failed to get URL: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v\n", err)
		return fmt.Errorf("failed to read response body: %w", err)
	}

	bodyStr := string(body)
	imagePattern := "(?i)https://[A-Za-z][0-9]+-([A-Za-z]+(-[A-Za-z]+)+)[0-9]+\\.tiktokcdn-us\\.com/"
	videoPattern := `https:\/\/v19-webapp-prime\.tiktok\.com\/.*$`
	re := regexp.MustCompile(imagePattern)
	revideo := regexp.MustCompile(videoPattern)

	detectedURLs := re.FindAllString(bodyStr, -1)
	detectedVideo := revideo.FindAllString(bodyStr, -1)
	fmt.Println(detectedURLs)
	fmt.Println(detectedVideo)
	var message string
	if len(detectedURLs) > 0 {
		message = "Detected TikTok CDN URLs:\n" + strings.Join(detectedURLs, "\n")
	} else if len(detectedVideo) > 0 {
		message = "Video found:\n" + strings.Join(detectedVideo, "\n")
	} else {
		message = "Nothing was found"
	}

	_, err = s.FollowupMessageCreate(m.Interaction, true, &discordgo.WebhookParams{
		Content: message,
	})
	if err != nil {
		fmt.Println("error creating followup message")
	}
	return nil
}
