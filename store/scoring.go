package store

type resolvedResult struct {
	result WordleResult
	name   string // display name after resolution via Resolver
}

func computeAverages(results []resolvedResult) map[string]float64 {
	scores := map[string][]int{}
	for _, r := range results {
		if !r.result.Complete {
			continue
		}
		scores[r.name] = append(scores[r.name], r.result.Score)
	}

	avgs := make(map[string]float64, len(scores))
	for name, ss := range scores {
		sum := 0
		for _, s := range ss {
			sum += s
		}
		avgs[name] = float64(sum) / float64(len(ss))
	}
	return avgs
}
