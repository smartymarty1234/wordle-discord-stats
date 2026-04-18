package store

import (
	"math"
	"testing"
)

var scenarioResults = []WordleResult{
	{UserID: "alice", Day: 1, Score: 2, Complete: true},
	{UserID: "bob", Day: 1, Score: 4, Complete: true},
	{UserID: "alice", Day: 2, Score: 3, Complete: true},
	{UserID: "bob", Day: 3, Score: 5, Complete: true},
}

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func identity(id string) string { return id }

func TestScenario_Average(t *testing.T) {
	avgs := computeAverages(scenarioResults, ScoringAverage, identity)
	if !approx(avgs["alice"], 2.5) {
		t.Errorf("alice: got %.4f, want 2.5", avgs["alice"])
	}
	if !approx(avgs["bob"], 4.5) {
		t.Errorf("bob: got %.4f, want 4.5", avgs["bob"])
	}
}

func TestScenario_Bayesian(t *testing.T) {
	// globalMean = (2+3+4+5)/4 = 3.5, C=10
	// alice: (10*3.5 + 5) / 12 = 40/12
	// bob:   (10*3.5 + 9) / 12 = 44/12
	avgs := computeAverages(scenarioResults, ScoringBayesianWeighted, identity)
	if !approx(avgs["alice"], 40.0/12.0) {
		t.Errorf("alice: got %.4f, want %.4f", avgs["alice"], 40.0/12.0)
	}
	if !approx(avgs["bob"], 44.0/12.0) {
		t.Errorf("bob: got %.4f, want %.4f", avgs["bob"], 44.0/12.0)
	}
}

func TestFixedNickMerge(t *testing.T) {
	// "alice" plays as a snowflake on day 1, as a fixed nick on day 2.
	// With a resolver that maps both to "alice", they should be merged.
	results := []WordleResult{
		{UserID: "snowflake-alice", Day: 1, Score: 2, Complete: true},
		{UserID: "@alice", Day: 2, Score: 4, Complete: true},
	}
	resolve := func(id string) string {
		if id == "snowflake-alice" {
			return "alice"
		}
		if id == "@alice" {
			return "alice"
		}
		return id
	}
	avgs := computeAverages(results, ScoringAverage, resolve)
	if !approx(avgs["alice"], 3.0) {
		t.Errorf("merged alice: got %.4f, want 3.0", avgs["alice"])
	}
	if len(avgs) != 1 {
		t.Errorf("expected 1 identity, got %d", len(avgs))
	}
}
