package daemon

import (
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"wordle-discord-stats/store"

	"github.com/bwmarrin/discordgo"
)

var allMentionRe = regexp.MustCompile(`<@(\d+)>|@([^<\n@>]+)`)

type userScore struct {
	userID    string
	fixedNick string
	score     int
	complete  bool
}

// parseMessage extracts WordleResults from an aggregate Wordle bot message.
// Returns nil, nil for non-aggregate messages.
func parseMessage(msg *discordgo.Message, imgparseBin string) ([]*store.WordleResult, error) {
	if !strings.Contains(msg.Content, "Here are yesterday's results") {
		return nil, nil
	}

	textScores := parseAggregateScores(msg.Content)
	if len(textScores) == 0 {
		slog.Debug("daemon: skip msg no scores parsed", "id", msg.ID)
		return nil, nil
	}

	slog.Info("daemon: found aggregate message", "id", msg.ID, "players", len(textScores))

	imageURL := imageURLFromMessage(msg)
	if imageURL == "" {
		return nil, fmt.Errorf("msg=%s no image attached", msg.ID)
	}

	day, err := extractWordleDay(imgparseBin, imageURL)
	if err != nil {
		return nil, fmt.Errorf("msg=%s imgparse: %w", msg.ID, err)
	}

	results := make([]*store.WordleResult, len(textScores))
	for i, us := range textScores {
		results[i] = &store.WordleResult{
			GuildID:   msg.GuildID,
			UserID:    us.userID,
			FixedNick: us.fixedNick,
			MessageID: msg.ID,
			Day:       day,
			Score:     us.score,
			Complete:  us.complete,
		}
	}
	return results, nil
}

func parseAggregateScores(content string) []userScore {
	var results []userScore
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		slashIdx := strings.Index(line, "/6:")
		if slashIdx < 1 {
			continue
		}
		prefix := strings.TrimSpace(line[:slashIdx])

		var score int
		var complete bool
		last := prefix[len(prefix)-1]
		if last == 'X' || last == 'x' {
			score, complete = 0, false
		} else {
			i := len(prefix)
			for i > 0 && prefix[i-1] >= '0' && prefix[i-1] <= '9' {
				i--
			}
			n, err := strconv.Atoi(prefix[i:])
			if err != nil || n < 1 || n > 6 {
				continue
			}
			score, complete = n, true
		}

		after := line[slashIdx+3:]
		for _, m := range allMentionRe.FindAllStringSubmatch(after, -1) {
			if m[1] != "" {
				results = append(results, userScore{userID: m[1], score: score, complete: complete})
			} else {
				name := strings.TrimSpace(m[2])
				results = append(results, userScore{fixedNick: name, score: score, complete: complete})
			}
		}
	}
	return results
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

// extractWordleDay calls the imgparse binary with the image URL and returns
// the Wordle day number.
func extractWordleDay(bin, imageURL string) (int, error) {
	out, err := exec.Command(bin, imageURL).Output()
	if err != nil {
		return 0, err
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return 0, fmt.Errorf("imgparse returned empty output for %s", imageURL)
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("imgparse returned non-numeric output %q", s)
	}
	return n, nil
}
