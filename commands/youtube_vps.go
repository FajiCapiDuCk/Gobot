//go:build VPS

package commands
import "os/exec"
func Youtubetaker(name string, url string) *exec.Cmd  {
	return exec.Command("yt-dlp", "-f", "18", "--cookies", "youtube_cookies.txt", "-o", name, url)
}
