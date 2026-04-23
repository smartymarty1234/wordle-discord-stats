package daemon

import (
	"log/slog"
	"os"
	"sort"
	"sync"
	"time"

	"wordle-discord-stats/store"

	"github.com/bwmarrin/discordgo"
)

const (
	batchSize    = 100
	pollInterval = 10 * time.Minute
)

// Resolver maps a player key (snowflake or fixed nick) to a display name.
type Resolver interface {
	Get(key string) string
}

type Daemon struct {
	session     *discordgo.Session
	channelID   string
	botUserID   string
	imgparseBin string
	cursor      MessageCursor
	store       store.Store
	resolver    Resolver
}

func New(session *discordgo.Session, channelID, botUserID, imgparseBin string, cursor MessageCursor, st store.Store, resolver Resolver) *Daemon {
	return &Daemon{
		session:     session,
		channelID:   channelID,
		botUserID:   botUserID,
		imgparseBin: imgparseBin,
		cursor:      cursor,
		store:       st,
		resolver:    resolver,
	}
}

func (d *Daemon) Run() {
	d.ingest()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for range ticker.C {
		d.ingest()
	}
}

func (d *Daemon) postReport() {
	entries, err := d.store.QueryTop(5, 0)
	if err != nil {
		slog.Error("postReport: query", "err", err)
		return
	}
	msg := buildHeader(time.Now()) + "**Top 5 (all time)**\n" + store.FormatTop(entries)
	if _, err := d.session.ChannelMessageSend(d.channelID, msg); err != nil {
		slog.Error("postReport: send", "err", err)
	}
}

func (d *Daemon) ingest() {
	afterID, err := d.cursor.Get()
	switch {
	case os.IsNotExist(err):
		slog.Info("daemon: no cursor, starting from beginning")
		afterID = "0"
	case err != nil:
		slog.Error("daemon: cursor get", "err", err)
		return
	}

	var savedNew bool
	for {
		slog.Info("daemon: fetching batch", "after", afterID)

		msgs, err := d.session.ChannelMessages(d.channelID, batchSize, "", afterID, "")
		if err != nil {
			slog.Error("daemon: fetch messages", "err", err)
			return
		}
		if len(msgs) == 0 {
			break
		}

		sort.Slice(msgs, func(i, j int) bool {
			return msgs[i].Timestamp.Before(msgs[j].Timestamp)
		})

		var botMsgs []*discordgo.Message
		for _, msg := range msgs {
			if msg.Author != nil && msg.Author.ID == d.botUserID {
				botMsgs = append(botMsgs, msg)
			}
		}
		slog.Info("daemon: batch filtered", "total", len(msgs), "bot", len(botMsgs))

		type parseResult struct {
			msg     *discordgo.Message
			results []*store.WordleResult
			err     error
		}
		ch := make(chan parseResult, len(botMsgs))
		var wg sync.WaitGroup
		for _, msg := range botMsgs {
			wg.Add(1)
			go func(msg *discordgo.Message) {
				defer wg.Done()
				results, err := parseMessage(msg, d.imgparseBin)
				ch <- parseResult{msg, results, err}
			}(msg)
		}
		wg.Wait()
		close(ch)

		for pr := range ch {
			if pr.err != nil {
				slog.Error("daemon: parse message", "msg", pr.msg.ID, "err", pr.err)
				continue
			}
			for _, result := range pr.results {
				isNew, err := d.store.Save(*result)
				if err != nil {
					slog.Error("daemon: save", "err", err)
				} else if isNew {
					savedNew = true
					slog.Info("daemon: saved", "day", result.Day, "user", result.UserID, "fixed_nick", result.FixedNick, "score", result.Score)
				}
			}
		}

		maxID := msgs[0].ID
		for _, m := range msgs[1:] {
			if m.ID > maxID {
				maxID = m.ID
			}
		}
		afterID = maxID
		if err := d.cursor.Set(maxID); err != nil {
			slog.Error("daemon: cursor set", "err", err)
		}
	}

	slog.Info("daemon: ingest done", "next_poll", pollInterval)
	if savedNew {
		d.postReport()
	}
}
