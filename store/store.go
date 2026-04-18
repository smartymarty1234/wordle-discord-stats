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
	QueryStats(userID string, sinceDay int, scoringType ScoringType) (StatsResult, error)
	QueryTop(k int, sinceDay int, scoringType ScoringType) ([]TopEntry, error)
	UserIDs() ([]string, error)
}

func FormatTop(entries []TopEntry, resolve func(string) string) string {
	msg := ""
	for i, e := range entries {
		msg += fmt.Sprintf("%d. %s — %.2f\n", i+1, resolve(e.UserID), e.AvgScore)
	}
	return msg
}
