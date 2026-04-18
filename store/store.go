package store

import "fmt"

type WordleResult struct {
	GuildID   string
	UserID    string
	FixedNick string
	MessageID string
	Day       int
	Score     int
	Complete  bool
}

type StatsResult struct {
	AvgScore float64
	Rank     int
}

type TopEntry struct {
	UserID   string
	AvgScore float64
}

type ScoringType string

const (
	ScoringAverage          ScoringType = "average"
	ScoringBayesianWeighted ScoringType = "bayesian_weighted"
)

type Store interface {
	Save(result WordleResult) (bool, error)
	// resolveIdentity maps a stored UserID to a canonical display-name key,
	// allowing fixed-nick and snowflake results for the same person to be merged.
	QueryStats(userID string, sinceDay int, scoringType ScoringType, resolveIdentity func(string) string) (StatsResult, error)
	QueryTop(k int, sinceDay int, scoringType ScoringType, resolveIdentity func(string) string) ([]TopEntry, error)
	UserIDs() ([]string, error)
}

// FormatTop formats a leaderboard. TopEntry.UserID is already a resolved
// display-name identity (set by QueryTop), so no separate resolver is needed.
func FormatTop(entries []TopEntry) string {
	msg := ""
	for i, e := range entries {
		msg += fmt.Sprintf("%d. %s — %.2f\n", i+1, e.UserID, e.AvgScore)
	}
	return msg
}
