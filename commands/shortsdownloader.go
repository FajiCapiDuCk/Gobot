package commands

import (
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/bwmarrin/discordgo"
)

const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func StringWithCharset(length int, charset string) string {
	r := rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), uint64(time.Now().UnixNano())))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[r.IntN(len(charset))]
	}
	return string(b)
}

func isTiktokDomain(url string) bool {
	pattern := `^(https?:\/\/)?([a-zA-Z0-9-]+\.)*tiktok\.com\/?`
	re := regexp.MustCompile(pattern)
	return re.MatchString(url)
}

func isYoutubeShorts(url string) bool {
	pattern := `https://(www\.)?youtube\.com/shorts/.*`
	re := regexp.MustCompile(pattern)
	return re.MatchString(url)
}

func Stealshorts(s *discordgo.Session, url string, i *discordgo.InteractionCreate) {
	var message string
	if !isTiktokDomain(url) && !isYoutubeShorts(url) {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "URL must be from tiktok or youtube shorts",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			fmt.Printf("ERROR: Unable to send reponse to wrong link\n%s", err)
		}
		return
	}
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("Error sending thinking message: %v\n", err)
		return
	}
	tmp_name := StringWithCharset(20, charset)
	full_file_name := fmt.Sprintf("/tmp/%s.mp4", tmp_name)

	if isTiktokDomain(url) {
		re := regexp.MustCompile("(?i)h264_[A-Za-z0-9]+_[A-Za-z0-9]+-[A-Za-z0-9]+")
		tiktokcheckcmd := exec.Command("yt-dlp", "-F", url)
		output, err := tiktokcheckcmd.Output()
		quality_parameter := re.FindString(string(output))
		tiktokstealcmd := exec.Command("yt-dlp", "-f", quality_parameter, "-o", full_file_name, url)
		err = tiktokstealcmd.Start()
		if err != nil {
			fmt.Printf("lol ha funyn how happened %s\n", err)
		}
		err = tiktokstealcmd.Wait()
		if err != nil {
			message = "Images arent supported"
			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &message,
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			return
		}
	} else {
		youtubestealcmd := exec.Command("yt-dlp", "-f", "18", "-o", full_file_name, url)
		err := youtubestealcmd.Start()
		if err != nil {
			fmt.Printf("interesting %s", err)
		}
		err = youtubestealcmd.Wait()
		if err != nil {
			fmt.Println(err)
		}
	}
	file, err := os.Open(full_file_name)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
	}
	_, err = s.ChannelFileSend(i.ChannelID, full_file_name, file)
	message = fmt.Sprintf("File size was too big to send to discord, consider using yt-dlp to install it manually yourself\n Here is the link: %s", url)
	if err != nil {
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &message,
		})
		return
	}
	defer os.Remove(full_file_name)
	message = "dont read this :)"
	messagesent, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &message,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	s.ChannelMessageDelete(messagesent.ChannelID, messagesent.ID)
	if err != nil {
		fmt.Printf("ERROR: Unable to edit good send video message\n%s", err)
	}
}
