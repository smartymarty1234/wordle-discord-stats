package store

import (
	"math"
	"sort"
)

type resolvedResult struct {
	result WordleResult
	name   string // display name after resolution via Resolver
}

// avgPerPlayer computes the mean score for each player. Players with fewer
// than minGames results are excluded; minGames=0 disables the filter.
func avgPerPlayer(players map[string][]resolvedResult, minGames int) map[string]float64 {
	out := make(map[string]float64, len(players))
	for name, rs := range players {
		if len(rs) < minGames {
			continue
		}
		sum := 0
		for _, r := range rs {
			sum += r.result.Score
		}
		out[name] = float64(sum) / float64(len(rs))
	}
	return out
}

// currentStreaks returns the consecutive-days-played count per player,
// ending exactly at latest. A player whose most recent result isn't
// latest has a current streak of 0. Input slices must be day-ascending.
func currentStreaks(players map[string][]resolvedResult, latest int) map[string]float64 {
	out := make(map[string]float64, len(players))
	for name, rs := range players {
		if latest == 0 || len(rs) == 0 || rs[len(rs)-1].result.Day != latest {
			out[name] = 0
			continue
		}
		streak := 1
		for i := len(rs) - 2; i >= 0; i-- {
			if rs[i].result.Day == rs[i+1].result.Day-1 {
				streak++
				continue
			}
			break
		}
		out[name] = float64(streak)
	}
	return out
}

// allTimeStreaks returns, for each player, the longest run of consecutive
// days played and the final day of that run. Input slices must be day-ascending.
func allTimeStreaks(players map[string][]resolvedResult) []Entry {
	entries := make([]Entry, 0, len(players))
	for name, rs := range players {
		if len(rs) == 0 {
			continue
		}
		best, bestEnd := 1, rs[0].result.Day
		cur := 1
		for i := 1; i < len(rs); i++ {
			if rs[i].result.Day == rs[i-1].result.Day+1 {
				cur++
			} else {
				cur = 1
			}
			if cur > best {
				best, bestEnd = cur, rs[i].result.Day
			}
		}
		entries = append(entries, Entry{Name: name, Value: float64(best), Day: bestEnd})
	}
	return entries
}

func scoresAtMost(players map[string][]resolvedResult, threshold int) map[string]float64 {
	out := make(map[string]float64, len(players))
	for name, rs := range players {
		count := 0
		for _, r := range rs {
			if r.result.Score <= threshold {
				count++
			}
		}
		out[name] = float64(count)
	}
	return out
}

// totalElo runs long-term Elo across all days in chronological order. Each
// day is played out as a round-robin of 1v1 matches between every pair of
// players that played that day; the lower score wins (ties split). Ratings
// carry over between days; unseen players enter at start.
func totalElo(days map[int][]resolvedResult, start, k float64) map[string]float64 {
	dayNums := make([]int, 0, len(days))
	for d := range days {
		dayNums = append(dayNums, d)
	}
	sort.Ints(dayNums)

	ratings := map[string]float64{}
	for _, d := range dayNums {
		entries := days[d]
		for _, r := range entries {
			if _, ok := ratings[r.name]; !ok {
				ratings[r.name] = start
			}
		}
		for i := 0; i < len(entries); i++ {
			for j := i + 1; j < len(entries); j++ {
				a, b := entries[i], entries[j]
				var sA float64
				switch {
				case a.result.Score < b.result.Score:
					sA = 1
				case a.result.Score > b.result.Score:
					sA = 0
				default:
					sA = 0.5
				}
				rA, rB := ratings[a.name], ratings[b.name]
				eA := 1 / (1 + math.Pow(10, (rB-rA)/400))
				ratings[a.name] = rA + k*(sA-eA)
				ratings[b.name] = rB + k*((1-sA)-(1-eA))
			}
		}
	}
	return ratings
}
