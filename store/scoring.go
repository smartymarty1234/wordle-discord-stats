package store

type scoredEntry struct {
	userID string
	day    int
	score  int
}

// bayesianC is the number of virtual games added at the global mean.
// Higher values pull sparse users closer to the mean.
const bayesianC = 10

func computeAverages(results []WordleResult, scoringType ScoringType) map[string]float64 {
	entries := completedEntries(results)

	userScores := map[string][]int{}
	for _, e := range entries {
		userScores[e.userID] = append(userScores[e.userID], e.score)
	}

	if scoringType != ScoringBayesianWeighted {
		avgs := make(map[string]float64, len(userScores))
		for uid, scores := range userScores {
			sum := 0
			for _, s := range scores {
				sum += s
			}
			avgs[uid] = float64(sum) / float64(len(scores))
		}
		return avgs
	}

	var globalSum float64
	for _, e := range entries {
		globalSum += float64(e.score)
	}
	globalMean := globalSum / float64(len(entries))

	bayesian := make(map[string]float64, len(userScores))
	for uid, scores := range userScores {
		n := float64(len(scores))
		var sum float64
		for _, s := range scores {
			sum += float64(s)
		}
		bayesian[uid] = (bayesianC*globalMean + sum) / (bayesianC + n)
	}
	return bayesian
}

func completedEntries(results []WordleResult) []scoredEntry {
	entries := make([]scoredEntry, 0, len(results))
	for _, r := range results {
		if !r.Complete {
			continue
		}
		id := r.UserID
		if r.FixedNick != "" {
			id = r.FixedNick
		}
		entries = append(entries, scoredEntry{id, r.Day, r.Score})
	}
	return entries
}
