package store

import "fmt"

type WordleResult struct {
	GuildID   string
	UserID    string // Discord snowflake; empty if FixedNick is set
	FixedNick string // non-Discord player name; empty if UserID is set
	MessageID string
	Day       int
	Score     int
	Complete  bool
}

// PlayerKey returns the raw identity key for a result before resolution.
// UserID results require nickcache resolution; FixedNick results are already display names.
func PlayerKey(r WordleResult) string {
	if r.FixedNick != "" {
		return r.FixedNick
	}
	return r.UserID
}

// Resolver maps a player key to a display name.
// nickcache.Get satisfies this: snowflakes resolve to their guild nick,
// and fixed nicks pass through unchanged (not in the cache).
type Resolver interface {
	Get(key string) string
}

type StatsResult struct {
	AvgScore float64
	Rank     int
}

type TopEntry struct {
	Name     string // resolved display name
	AvgScore float64
}

type Store interface {
	Save(result WordleResult) (bool, error)
	QueryStats(playerKey string, sinceDay int) (StatsResult, error)
	QueryTop(k int, sinceDay int) ([]TopEntry, error)
}

func FormatTop(entries []TopEntry) string {
	msg := ""
	for i, e := range entries {
		msg += fmt.Sprintf("%d. %s — %.2f\n", i+1, e.Name, e.AvgScore)
	}
	return msg
}
