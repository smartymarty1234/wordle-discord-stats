// gen_fixtures pages through a Discord channel, finds aggregate Wordle bot
// messages, downloads their image attachments, runs imgparse to determine
// the wordle day number, and writes each image to imgparse/tests/<day>.png.
//
// Required env vars: DISCORD_TOKEN, DISCORD_CHANNEL_ID, WORDLE_BOT_USER_ID
// Optional:          IMGPARSE_BIN (default: ./imgparse/target/debug/imgparse)
//                    OUT_DIR      (default: ./imgparse/tests)
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

const (
	batchSize    = 100
	aggregateKey = "Here are yesterday's results"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

func main() {
	token := mustEnv("DISCORD_TOKEN")
	channelID := mustEnv("DISCORD_CHANNEL_ID")
	botUserID := mustEnv("WORDLE_BOT_USER_ID")

	imgparseBin := envOr("IMGPARSE_BIN", "./imgparse/target/debug/imgparse")
	outDir := envOr("OUT_DIR", "./imgparse/tests")

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("mkdir %s: %v", outDir, err)
	}

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("discordgo.New: %v", err)
	}

	afterID := "1413451698722308208"
	saved, skipped := 0, 0

	for {
		msgs, err := s.ChannelMessages(channelID, batchSize, "", afterID, "")
		if err != nil {
			log.Fatalf("ChannelMessages: %v", err)
		}
		if len(msgs) == 0 {
			break
		}

		sort.Slice(msgs, func(i, j int) bool {
			return msgs[i].Timestamp.Before(msgs[j].Timestamp)
		})

		for _, msg := range msgs {
			if msg.Author == nil || msg.Author.ID != botUserID {
				continue
			}
			if !strings.Contains(msg.Content, aggregateKey) {
				continue
			}

			imageURL := imageURLFromMessage(msg)
			if imageURL == "" {
				log.Printf("skip msg=%s: no image attachment", msg.ID)
				skipped++
				continue
			}

			t0 := time.Now()
			day, err := runImgparse(imgparseBin, imageURL)
			imgparseElapsed := time.Since(t0)
			if err != nil {
				log.Printf("skip msg=%s: imgparse (%s): %v", msg.ID, imgparseElapsed, err)
				skipped++
				continue
			}
			if day == "" {
				log.Printf("skip msg=%s: imgparse (%s) returned empty day", msg.ID, imgparseElapsed)
				skipped++
				continue
			}

			dest := filepath.Join(outDir, day+".png")
			if _, err := os.Stat(dest); err == nil {
				log.Printf("skip msg=%s: %s already exists (imgparse=%s)", msg.ID, dest, imgparseElapsed)
				skipped++
				continue
			}

			t1 := time.Now()
			if err := downloadImage(imageURL, dest); err != nil {
				log.Printf("skip msg=%s: download (%s): %v", msg.ID, time.Since(t1), err)
				skipped++
				continue
			}
			downloadElapsed := time.Since(t1)

			log.Printf("saved day=%s msg=%s imgparse=%s download=%s", day, msg.ID, imgparseElapsed, downloadElapsed)
			saved++
		}

		maxID := msgs[0].ID
		for _, m := range msgs[1:] {
			if m.ID > maxID {
				maxID = m.ID
			}
		}
		afterID = maxID
	}

	fmt.Printf("done: saved=%d skipped=%d\n", saved, skipped)
}

func imageURLFromMessage(msg *discordgo.Message) string {
	for _, a := range msg.Attachments {
		return a.URL
	}
	for _, e := range msg.Embeds {
		if e.Image != nil {
			return e.Image.URL
		}
	}
	return ""
}

// runImgparse invokes the imgparse binary with the image URL and returns the
// day number as a trimmed string.
func runImgparse(bin, imageURL string) (string, error) {
	out, err := exec.Command(bin, imageURL).Output()
	if err != nil {
		return "", fmt.Errorf("%w", err)
	}
	day := strings.TrimSpace(string(out))
	if day == "" {
		return "", nil
	}
	return day, nil
}

func downloadImage(url, dest string) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s not set", key)
	}
	return v
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
