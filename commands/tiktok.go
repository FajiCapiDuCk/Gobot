package commands

import (
	"errors"
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

func isTiktokDomain(u string) bool {
	pattern := `^(https?:\/\/)?([a-zA-Z0-9-]+\.)*tiktok\.com\/?`
	re := regexp.MustCompile(pattern)
	return re.MatchString(u)
}

func Processtiktok(s *discordgo.Session, tiktokURL string, i *discordgo.InteractionCreate) error {
	var message string
	if !isTiktokDomain(tiktokURL) {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "URL must be tiktok",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			fmt.Printf("ERROR: Unable to send reponse to wrong link\n%s", err)
		}
		return errors.New("Some moron sent not tiktok all good")
	}
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("Error sending thinking message: %v\n", err)
		return err
	}

	re := regexp.MustCompile("(?i)h264_[A-Za-z0-9]+_[A-Za-z0-9]+-[A-Za-z0-9]+")
	tiktokcheckcmd := exec.Command("yt-dlp", "-F", tiktokURL)
	output, err := tiktokcheckcmd.Output()
	quality_parameter := re.FindString(string(output))
	tmp_name := StringWithCharset(20, charset)
	full_file_name := fmt.Sprintf("/tmp/%s.mp4", tmp_name)
	tiktokstealcmd := exec.Command("yt-dlp", "-f", quality_parameter, "-o", full_file_name, tiktokURL)
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
		return err
	}
	file, err := os.Open(full_file_name)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
	}
	_, err = s.ChannelFileSend(i.ChannelID, full_file_name, file)
	message = fmt.Sprintf("File size was too big to send to discord, consider using yt-dlp to install it manually yourself\n Here is the link: %s", tiktokURL)
	if err != nil {
		_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &message,
		})
	}
	os.Remove(full_file_name)
	message = "dont read this :)"
	messagesent, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &message,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	s.ChannelMessageDelete(messagesent.ChannelID, messagesent.ID)
	if err != nil {
		fmt.Printf("ERROR: Unable to edit good send video message\n%s", err)
	}
	return nil
}
