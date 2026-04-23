package store

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
