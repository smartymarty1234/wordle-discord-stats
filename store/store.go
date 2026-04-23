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

// QueryKind selects which scoring feature to compute. Each kind produces a
// player → value mapping; the selector then picks top-k, bottom-k, or a
// single player from that mapping.
type QueryKind int

const (
	// KindAvgAllTime: average score per player across all time.
	// Params: MinGames (exclude players with fewer games; 0 = no filter).
	KindAvgAllTime QueryKind = iota
)

// Selector picks which subset of the player→value mapping to return.
type Selector int

const (
	SelectorTopK    Selector = iota // lowest values first (best Wordle scores)
	SelectorBottomK                 // highest values first
	SelectorPlayer                  // single player, identified by Query.Player
)

type Query struct {
	Kind     QueryKind
	Selector Selector

	K      int    // for TopK / BottomK
	Player string // player key (snowflake or fixed nick) for SelectorPlayer

	// Feature-specific parameters. Zero values are allowed and mean "no
	// filter" / sensible default where applicable.

	MinGames int // KindAvgAllTime
}

// Entry is one row of a query result.
type Entry struct {
	Name  string  // resolved display name
	Value float64 // the computed value for this feature
	Rank  int     // 1-based rank among all matching players (filled for SelectorPlayer)
}

type QueryResult struct {
	Entries []Entry
}

type Store interface {
	Save(result WordleResult) (bool, error)
	Query(q Query) (QueryResult, error)
}

func FormatEntries(entries []Entry) string {
	msg := ""
	for i, e := range entries {
		msg += fmt.Sprintf("%d. %s — %.2f\n", i+1, e.Name, e.Value)
	}
	return msg
}
