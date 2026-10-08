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

func isInstragram(url string) bool {
	pattern := `https://www\.instagram\.com/reel/.*`
	re := regexp.MustCompile(pattern)
	return re.MatchString(url)
}

func Stealshorts(s *discordgo.Session, url string, i *discordgo.InteractionCreate) {
	var message string
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	if err != nil {
		log.Printf("Error sending thinking message: %v\n", err)
		return
	}
	
	full_file_name := fmt.Sprintf("/tmp/%s.mp4", StringWithCharset(20, charset))
	full_audio_name := fmt.Sprintf("/tmp/%s.mp4", StringWithCharset(20, charset))
	full_second_name := fmt.Sprintf("/tmp/%s.mp4", StringWithCharset(20, charset))

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
			})
			return
		}
	} else if isYoutubeShorts(url) {
		// If machine has residental IP needs to have youtube_cookies.txt at the same place where executable is running
		// https://github.com/yt-dlp/yt-dlp/wiki/FAQ#how-do-i-pass-cookies-to-yt-dlp
		youtubestealcmd := Youtubetaker(full_file_name, url)
		err := youtubestealcmd.Start()
		if err != nil {
			fmt.Printf("Error starting youtube stealer: %s", err)
		}
		err = youtubestealcmd.Wait()
		if err != nil {
			fmt.Println(err)
		} 
		} else if isInstragram(url) {
		re := regexp.MustCompile("(?i)dash-[0-9]+v")
		checkcmd := exec.Command("yt-dlp", "-F", url)
		output, _ := checkcmd.Output()
		quality_parameter := re.Find(output)
		stealcmd := exec.Command("yt-dlp", "-f", string(quality_parameter), "-o", full_second_name, url)
		err = stealcmd.Start()
		if err != nil {
			fmt.Printf("Instragram stealer failed to start: %s\n", err)
		}
		err = stealcmd.Wait()
		if err != nil {
			message = "Instragram failed to download"
			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &message,
			})
			return
		}
		// Audio part of the instagram because it is retarded and stupid and ffmpeg will be used to recombine the final video
		/* 
		ffmpeg -i input_video.mp4 \
		  -i input_audio.mp3 \
		  -c:v copy -c:a copy -map 0:v:0 -map 1:a:0 \
		  output.mp4
		*/
		re = regexp.MustCompile("(?i)dash-[0-9]+a")
		checkcmd = exec.Command("yt-dlp", "-F", url)
		output, _ = checkcmd.Output()
		quality_parameter = re.Find(output)
		
		stealcmd = exec.Command("yt-dlp", "-f", string(quality_parameter), "-o", full_audio_name, url)
		err = stealcmd.Start()
		if err != nil {
			fmt.Printf("Instragram stealer failed to start: %s\n", err)
		}
		err = stealcmd.Wait()
		if err != nil {
			message = "Instragram failed to download"
			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &message,
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			return
		}
		combine := exec.Command("ffmpeg", "-i", full_second_name, "-i", full_audio_name, "-c:v", "copy", "-c:a", "copy", full_file_name)
		err = combine.Start()
		if err != nil {
			fmt.Printf("Failed to start ffmpeg combiner for instragram: %s\n", err)
		}
		err = combine.Wait()
		if err != nil {
			fmt.Printf("Combiner failed: %s\n", err)
		}
		os.Remove(full_second_name)
		os.Remove(full_audio_name)
	} else {
		message = fmt.Sprint("Sent link was not any of accepted formats\nAccepted links are: Youtube, Tiktok and Instagram")
		if err != nil {
			_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &message,
			})
	}
	return
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
			Flags: discordgo.MessageFlagsEphemeral,
		})
		return
	}
	defer os.Remove(full_file_name)
	message = "dont read this :)"
	messagesent, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &message,
	})
	s.ChannelMessageDelete(messagesent.ChannelID, messagesent.ID)
	if err != nil {
		fmt.Printf("ERROR: Unable to edit good send video message\n%s", err)
	}
}
