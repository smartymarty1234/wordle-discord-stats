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

// DNFScore is the score assigned to a result where the player started but
// did not finish. Stored results keep their raw score (0 for DNF); load()
// applies this interpretation so scoring code sees a real number.
const DNFScore = 7

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
		if !r.Complete {
			r.Score = DNFScore
		}
		results = append(results, r)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Day != results[j].Day {
			return results[i].Day < results[j].Day
		}
		return PlayerKey(results[i]) < PlayerKey(results[j])
	})
	return results, nil
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

// Query computes a scoring feature and selects from the resulting
// player → value mapping per q.Selector.
func (f *FileStore) Query(q Query) (QueryResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	entries, err := f.computeEntries(q)
	if err != nil {
		return QueryResult{}, err
	}
	sortEntries(entries, q.Kind)

	switch q.Selector {
	case SelectorTopK:
		if q.K < len(entries) {
			entries = entries[:q.K]
		}
	case SelectorBottomK:
		entries = reverse(entries)
		if q.K < len(entries) {
			entries = entries[:q.K]
		}
	case SelectorPlayer:
		name := q.Player
		if f.resolver != nil {
			name = f.resolver.Get(q.Player)
		}
		for i, e := range entries {
			if e.Name == name {
				e.Rank = i + 1
				return QueryResult{Entries: []Entry{e}}, nil
			}
		}
		return QueryResult{}, fmt.Errorf("no results for %s", name)
	}
	return QueryResult{Entries: entries}, nil
}

// computeEntries dispatches to the feature-specific calculator for q.Kind.
func (f *FileStore) computeEntries(q Query) ([]Entry, error) {
	switch q.Kind {
	case KindAvgAllTime:
		players, err := f.perPlayer()
		if err != nil {
			return nil, err
		}
		return valuesToEntries(avgPerPlayer(players, q.MinGames)), nil
	case KindAvgSliding:
		players, err := f.perPlayerSince(latestDaySince(q.SlidingDays, f))
		if err != nil {
			return nil, err
		}
		return valuesToEntries(avgPerPlayer(players, 0)), nil
	case KindTotalElo:
		days, err := f.perDay()
		if err != nil {
			return nil, err
		}
		return valuesToEntries(totalElo(days, q.EloStart, q.EloK)), nil
	case KindCurrentStreak:
		players, err := f.perPlayer()
		if err != nil {
			return nil, err
		}
		return valuesToEntries(currentStreaks(players, latestDay(f))), nil
	case KindAllTimeStreak:
		players, err := f.perPlayer()
		if err != nil {
			return nil, err
		}
		return allTimeStreaks(players), nil
	case KindScoresAtMost:
		players, err := f.perPlayer()
		if err != nil {
			return nil, err
		}
		return valuesToEntries(scoresAtMost(players, q.ScoreAtMost)), nil
	}
	return nil, fmt.Errorf("unknown query kind: %d", q.Kind)
}

func valuesToEntries(values map[string]float64) []Entry {
	entries := make([]Entry, 0, len(values))
	for name, v := range values {
		entries = append(entries, Entry{Name: name, Value: v})
	}
	return entries
}

// latestDay returns the max wordle day present in the store, or 0 if empty.
func latestDay(f *FileStore) int {
	results, err := f.load()
	if err != nil || len(results) == 0 {
		return 0
	}
	max := results[0].Day
	for _, r := range results[1:] {
		if r.Day > max {
			max = r.Day
		}
	}
	return max
}

// perPlayerSince is perPlayer restricted to results on or after sinceDay.
// sinceDay == 0 means no filter.
func (f *FileStore) perPlayerSince(sinceDay int) (map[string][]resolvedResult, error) {
	all, err := f.perPlayer()
	if err != nil {
		return nil, err
	}
	if sinceDay == 0 {
		return all, nil
	}
	out := map[string][]resolvedResult{}
	for name, rs := range all {
		for _, r := range rs {
			if r.result.Day >= sinceDay {
				out[name] = append(out[name], r)
			}
		}
	}
	return out, nil
}

// latestDaySince computes the "since day" cutoff for a window of the given
// size, relative to the max day seen in the store. Returns 0 (no cutoff)
// when days <= 0 or the store is empty.
func latestDaySince(days int, f *FileStore) int {
	if days <= 0 {
		return 0
	}
	max := latestDay(f)
	if max == 0 {
		return 0
	}
	return max - days + 1
}

// perPlayer groups resolved results by display name. Iteration order within
// each slice follows load()'s (day, key) ordering.
func (f *FileStore) perPlayer() (map[string][]resolvedResult, error) {
	results, err := f.load()
	if err != nil {
		return nil, err
	}
	out := map[string][]resolvedResult{}
	for _, rr := range f.resolveAll(results) {
		out[rr.name] = append(out[rr.name], rr)
	}
	return out, nil
}

// perDay groups resolved results by wordle day. Iteration order within each
// slice follows load()'s (day, key) ordering (so per-day slices are ordered
// by player key).
func (f *FileStore) perDay() (map[int][]resolvedResult, error) {
	results, err := f.load()
	if err != nil {
		return nil, err
	}
	out := map[int][]resolvedResult{}
	for _, rr := range f.resolveAll(results) {
		out[rr.result.Day] = append(out[rr.result.Day], rr)
	}
	return out, nil
}

func reverse(e []Entry) []Entry {
	for i, j := 0, len(e)-1; i < j; i, j = i+1, j-1 {
		e[i], e[j] = e[j], e[i]
	}
	return e
}

// sortEntries orders entries "best first" for the given kind. For averages
// lower is better; for other kinds (streaks, counts) higher is better.
func sortEntries(entries []Entry, kind QueryKind) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Value != entries[j].Value {
			if averageKind(kind) {
				return entries[i].Value < entries[j].Value
			}
			return entries[i].Value > entries[j].Value
		}
		return entries[i].Name < entries[j].Name
	})
}

// averageKind reports whether the feature's "better" direction is lower.
// Averages favor lower scores; Elo and count-style features favor higher.
func averageKind(k QueryKind) bool {
	switch k {
	case KindAvgAllTime, KindAvgSliding:
		return true
	}
	return false
}
