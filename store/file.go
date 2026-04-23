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
	path     string
	mu       sync.Mutex
	resolver Resolver
}

func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

func (f *FileStore) SetResolver(r Resolver) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resolver = r
}

func (f *FileStore) resolveName(r WordleResult) string {
	key := PlayerKey(r)
	if f.resolver != nil {
		return f.resolver.Get(key)
	}
	return key
}

func (f *FileStore) resolveAll(results []WordleResult) []resolvedResult {
	out := make([]resolvedResult, len(results))
	for i, r := range results {
		out[i] = resolvedResult{result: r, name: f.resolveName(r)}
	}
	return out
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
	incomingName := f.resolveName(result)
	for _, r := range f.resolveAll(results) {
		if r.result.Day == result.Day && r.name == incomingName {
			return false, nil
		}
	}
	return true, f.persist(append(results, result))
}

func (f *FileStore) QueryStats(playerKey string, sinceDay int) (StatsResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	results, err := f.since(sinceDay)
	if err != nil {
		return StatsResult{}, err
	}

	avgs := computeAverages(f.resolveAll(results))

	// playerKey may be a snowflake; resolve it to match the display-name-keyed avgs map.
	name := playerKey
	if f.resolver != nil {
		name = f.resolver.Get(playerKey)
	}
	userAvg, ok := avgs[name]
	if !ok {
		return StatsResult{}, fmt.Errorf("no results for %s", name)
	}

	ranked := make([]TopEntry, 0, len(avgs))
	for n, avg := range avgs {
		ranked = append(ranked, TopEntry{Name: n, AvgScore: avg})
	}
	sortEntries(ranked)

	rank := 1
	for _, e := range ranked {
		if e.Name == name {
			break
		}
		rank++
	}

	return StatsResult{AvgScore: userAvg, Rank: rank}, nil
}

func (f *FileStore) QueryTop(k int, sinceDay int) ([]TopEntry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	results, err := f.since(sinceDay)
	if err != nil {
		return nil, err
	}

	avgs := computeAverages(f.resolveAll(results))
	entries := make([]TopEntry, 0, len(avgs))
	for name, avg := range avgs {
		entries = append(entries, TopEntry{Name: name, AvgScore: avg})
	}
	sortEntries(entries)

	if k < len(entries) {
		entries = entries[:k]
	}
	return entries, nil
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
		return entries[i].Name < entries[j].Name
	})
}
