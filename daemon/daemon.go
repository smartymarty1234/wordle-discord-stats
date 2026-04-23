package daemon

import (
	"fmt"
	"log/slog"
	"math/rand"
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

	// Scoring parameters fixed by the daemon; see docs/features.md.
	eloStart    = 1500.0
	eloK        = 32.0
	minGames    = 20
	slidingDays = 7
	topK        = 5
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
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	header := buildHeader(time.Now(), currentStreaksLine(d.store), funFactLine(d.store, r))

	msg := header +
		d.topBlock(":crown: All-time Elo", store.Query{Kind: store.KindTotalElo, Selector: store.SelectorTopK, K: topK, EloStart: eloStart, EloK: eloK}) +
		d.topBlock(fmt.Sprintf(":hourglass: All-time average (min %d games)", minGames), store.Query{Kind: store.KindAvgAllTime, Selector: store.SelectorTopK, K: topK, MinGames: minGames}) +
		d.topBlock(fmt.Sprintf(":clock1: %d-day sliding average", slidingDays), store.Query{Kind: store.KindAvgSliding, Selector: store.SelectorTopK, K: topK, SlidingDays: slidingDays})

	if _, err := d.session.ChannelMessageSend(d.channelID, msg); err != nil {
		slog.Error("postReport: send", "err", err)
	}
}

func (d *Daemon) topBlock(title string, q store.Query) string {
	res, err := d.store.Query(q)
	if err != nil {
		slog.Error("postReport: query", "title", title, "err", err)
		return ""
	}
	if len(res.Entries) == 0 {
		return fmt.Sprintf("**Top %d — %s**\n_(no eligible players yet)_\n\n", q.K, title)
	}
	return fmt.Sprintf("**Top %d — %s**\n%s\n", q.K, title, store.FormatEntries(res.Entries))
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
