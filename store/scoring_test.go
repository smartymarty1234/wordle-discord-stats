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

// Fixed-nick users have no UserID; their FixedNick is the identity key.
var fixedNickResults = []WordleResult{
	{FixedNick: "carol", Day: 1, Score: 3, Complete: true},
	{FixedNick: "carol", Day: 2, Score: 5, Complete: true},
}

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestScenario_Average(t *testing.T) {
	avgs := computeAverages(scenarioResults, ScoringAverage)
	if !approx(avgs["alice"], 2.5) {
		t.Errorf("alice: got %.4f, want 2.5", avgs["alice"])
	}
	if !approx(avgs["bob"], 4.5) {
		t.Errorf("bob: got %.4f, want 4.5", avgs["bob"])
	}
}

func TestFixedNick_Average(t *testing.T) {
	avgs := computeAverages(fixedNickResults, ScoringAverage)
	if !approx(avgs["carol"], 4.0) {
		t.Errorf("carol: got %.4f, want 4.0", avgs["carol"])
	}
	if _, ok := avgs[""]; ok {
		t.Error("empty-string key present; fixed-nick not keyed by FixedNick")
	}
}

func TestScenario_Bayesian(t *testing.T) {
	// globalMean = (2+3+4+5)/4 = 3.5, C=10
	// alice: (10*3.5 + 5) / 12 = 40/12
	// bob:   (10*3.5 + 9) / 12 = 44/12
	avgs := computeAverages(scenarioResults, ScoringBayesianWeighted)
	if !approx(avgs["alice"], 40.0/12.0) {
		t.Errorf("alice: got %.4f, want %.4f", avgs["alice"], 40.0/12.0)
	}
	if !approx(avgs["bob"], 44.0/12.0) {
		t.Errorf("bob: got %.4f, want %.4f", avgs["bob"], 44.0/12.0)
	}
}
