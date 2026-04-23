package daemon

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"wordle-discord-stats/store"
)

const headerGrid = "🟩 ⬛ ⬛ ⬛ ⬛\n🟩 🟩 🟨 ⬛ ⬛\n🟩 🟩 🟩 🟩 🟩"

// buildHeader assembles the report preamble: decorative grid, weekday title,
// current streaks line, and a single fun fact. streaks and funFact are
// included as present lines only when non-empty.
func buildHeader(t time.Time, streaks, funFact string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n**%s Stats**\n", headerGrid, t.Weekday())
	if streaks != "" {
		fmt.Fprintf(&b, "\n%s\n", streaks)
	}
	if funFact != "" {
		fmt.Fprintf(&b, "\n%s\n", funFact)
	}
	b.WriteString("\n")
	return b.String()
}

// currentStreaksLine queries current streaks and formats the non-zero ones,
// sorted best-first. Returns "" if no player is on a streak.
func currentStreaksLine(st store.Store) string {
	res, err := st.Query(store.Query{Kind: store.KindCurrentStreak, Selector: store.SelectorTopK, K: 1 << 30})
	if err != nil {
		return ""
	}
	var parts []string
	for _, e := range res.Entries {
		if e.Value <= 0 {
			break
		}
		parts = append(parts, fmt.Sprintf("%s (%d)", e.Name, int(e.Value)))
	}
	if len(parts) == 0 {
		return ""
	}
	return ":fire: **Current streaks:** " + strings.Join(parts, ", ")
}

// funFactLine picks one fun fact: 50/50 between the best all-time streak
// and a "scores ≤ x" count for a random player. Returns "" if neither is
// available.
func funFactLine(st store.Store, r *rand.Rand) string {
	if r.Intn(2) == 0 {
		if s := allTimeStreakFact(st); s != "" {
			return s
		}
		return scoresAtMostFact(st, r)
	}
	if s := scoresAtMostFact(st, r); s != "" {
		return s
	}
	return allTimeStreakFact(st)
}

func allTimeStreakFact(st store.Store) string {
	res, err := st.Query(store.Query{Kind: store.KindAllTimeStreak, Selector: store.SelectorTopK, K: 1})
	if err != nil || len(res.Entries) == 0 || res.Entries[0].Value < 2 {
		return ""
	}
	e := res.Entries[0]
	return fmt.Sprintf(":star: _Fun fact: %s's best streak was %d long, ended on wordle %d._", e.Name, int(e.Value), e.Day)
}

// scoresAtMostFact picks a random x in [2,4] and a random player whose count
// is non-zero at that threshold.
func scoresAtMostFact(st store.Store, r *rand.Rand) string {
	x := 2 + r.Intn(3)
	res, err := st.Query(store.Query{Kind: store.KindScoresAtMost, Selector: store.SelectorTopK, K: 1 << 30, ScoreAtMost: x})
	if err != nil {
		return ""
	}
	var eligible []store.Entry
	for _, e := range res.Entries {
		if e.Value <= 0 {
			break
		}
		eligible = append(eligible, e)
	}
	if len(eligible) == 0 {
		return ""
	}
	e := eligible[r.Intn(len(eligible))]
	return fmt.Sprintf(":star: _Fun fact: %s has scored ≤ %d on %d days._", e.Name, x, int(e.Value))
}
