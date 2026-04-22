package store

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
)

type FileStore struct {
	path string
	mu   sync.Mutex
}

func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

func (f *FileStore) load() ([]WordleResult, error) {
	file, err := os.Open(f.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var results []WordleResult
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var r WordleResult
		if err := json.Unmarshal(line, &r); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, scanner.Err()
}

func (f *FileStore) persist(results []WordleResult) error {
	var buf bytes.Buffer
	for _, r := range results {
		line, err := json.Marshal(r)
		if err != nil {
			return err
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	return os.WriteFile(f.path, buf.Bytes(), 0644)
}

func (f *FileStore) Save(result WordleResult) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	results, err := f.load()
	if err != nil {
		return false, err
	}
	for _, r := range results {
		if r.Day != result.Day {
			continue
		}
		if result.FixedNick != "" {
			if r.FixedNick == result.FixedNick {
				return false, nil
			}
		} else if r.UserID == result.UserID {
			return false, nil
		}
	}
	return true, f.persist(append(results, result))
}

func (f *FileStore) QueryStats(userID string, sinceDay int, scoringType ScoringType) (StatsResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	results, err := f.since(sinceDay)
	if err != nil {
		return StatsResult{}, err
	}

	avgs := computeAverages(results, scoringType)
	userAvg, ok := avgs[userID]
	if !ok {
		return StatsResult{}, fmt.Errorf("no results for %s", userID)
	}

	ranked := make([]TopEntry, 0, len(avgs))
	for uid, avg := range avgs {
		ranked = append(ranked, TopEntry{UserID: uid, AvgScore: avg})
	}
	sortEntries(ranked)

	rank := 1
	for _, e := range ranked {
		if e.UserID == userID {
			break
		}
		rank++
	}

	return StatsResult{AvgScore: userAvg, Rank: rank}, nil
}

func (f *FileStore) QueryTop(k int, sinceDay int, scoringType ScoringType) ([]TopEntry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	results, err := f.since(sinceDay)
	if err != nil {
		return nil, err
	}

	avgs := computeAverages(results, scoringType)
	entries := make([]TopEntry, 0, len(avgs))
	for uid, avg := range avgs {
		entries = append(entries, TopEntry{UserID: uid, AvgScore: avg})
	}
	sortEntries(entries)

	if k < len(entries) {
		entries = entries[:k]
	}
	return entries, nil
}

func (f *FileStore) UserIDs() ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	all, err := f.load()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var ids []string
	for _, r := range all {
		if r.FixedNick != "" {
			continue
		}
		if !seen[r.UserID] {
			seen[r.UserID] = true
			ids = append(ids, r.UserID)
		}
	}
	return ids, nil
}

func (f *FileStore) since(sinceDay int) ([]WordleResult, error) {
	all, err := f.load()
	if err != nil {
		return nil, err
	}
	if sinceDay == 0 {
		return all, nil
	}
	var out []WordleResult
	for _, r := range all {
		if r.Day >= sinceDay {
			out = append(out, r)
		}
	}
	return out, nil
}

func sortEntries(entries []TopEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].AvgScore != entries[j].AvgScore {
			return entries[i].AvgScore < entries[j].AvgScore
		}
		return entries[i].UserID < entries[j].UserID
	})
}
