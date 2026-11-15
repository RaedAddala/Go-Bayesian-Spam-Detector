package main

import (
	"math"
	"testing"
)

func TestNewModelVocabularyExcludesRareTokens(t *testing.T) {
	ham := Bag{"meeting": 5, "rare": 1}
	spam := Bag{"viagra": 4, "meeting": 1, "once": 1}

	m, err := NewModel(ham, spam, 3, 2, 3)
	if err != nil {
		t.Fatal(err)
	}
	if m.vocabSize != 2 {
		t.Errorf("vocabSize = %d, want 2 (meeting, viagra)", m.vocabSize)
	}
	if _, ok := m.TokenToID["rare"]; ok {
		t.Error("token below minCount should not be in the vocabulary")
	}
	// Totals cover only kept tokens: ham = meeting(5), spam = viagra(4) + meeting(1).
	if m.hamTokens != 5 || m.spamTokens != 5 {
		t.Errorf("hamTokens=%d spamTokens=%d, want 5 and 5", m.hamTokens, m.spamTokens)
	}

	tid := m.TokenToID["meeting"]
	if want := math.Log(6.0 / 7.0); math.Abs(m.logProbHam[tid]-want) > 1e-12 {
		t.Errorf("log P(meeting|ham) = %v, want %v", m.logProbHam[tid], want)
	}
	if want := math.Log(2.0 / 7.0); math.Abs(m.logProbSpam[tid]-want) > 1e-12 {
		t.Errorf("log P(meeting|spam) = %v, want %v", m.logProbSpam[tid], want)
	}
}

func TestNewModelRejectsEmptyClass(t *testing.T) {
	if _, err := NewModel(Bag{"abc": 1}, Bag{}, 1, 0, 1); err == nil {
		t.Error("expected an error when a class has no documents")
	}
}

func TestClassify(t *testing.T) {
	ham := Bag{"meeting": 10, "report": 10}
	spam := Bag{"viagra": 10, "winner": 10}
	m, err := NewModel(ham, spam, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Classify([]byte("Winner! Cheap VIAGRA")).Label; got != "spam" {
		t.Errorf("spammy text classified as %s", got)
	}
	if got := m.Classify([]byte("quarterly report before the meeting")).Label; got != "ham" {
		t.Errorf("hammy text classified as %s", got)
	}
}
