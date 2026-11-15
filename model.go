package main

import (
	"fmt"
	"math"
)

type Model struct {
	TokenToID  map[string]uint32
	HamCounts  []uint32
	SpamCounts []uint32

	// Totals and |V| cover only the kept vocabulary (tokens with
	// hamCount+spamCount >= minCount), so the Laplace-smoothed distribution
	// is over exactly the tokens that are scored.
	hamTokens  uint64
	spamTokens uint64
	vocabSize  uint32 // |V| for Laplace smoothing denominator

	// Precomputed log P(token | class), indexed by token ID.
	logProbHam  []float64
	logProbSpam []float64

	logPriorHam  float64
	logPriorSpam float64
}

// numHamDocs / numSpamDocs are document counts (emails).
// Tokens whose combined ham+spam count is below minCount are left out of the
// vocabulary entirely and are ignored at classification time.
func NewModel(hamBag, spamBag Bag, numHamDocs, numSpamDocs, minCount int) (*Model, error) {
	if numHamDocs <= 0 || numSpamDocs <= 0 {
		return nil, fmt.Errorf("need at least one document per class, got %d ham and %d spam", numHamDocs, numSpamDocs)
	}

	totalDocs := float64(numHamDocs + numSpamDocs)
	tokenToID := make(map[string]uint32)
	var hamCounts, spamCounts []uint32
	var hamTokens, spamTokens uint64

	keep := func(tok string, h, s int) {
		if h+s < minCount {
			return
		}
		tokenToID[tok] = uint32(len(hamCounts))
		hamCounts = append(hamCounts, uint32(h))
		spamCounts = append(spamCounts, uint32(s))
		hamTokens += uint64(h)
		spamTokens += uint64(s)
	}

	for tok, h := range hamBag {
		keep(tok, h, spamBag[tok])
	}
	for tok, s := range spamBag {
		if _, inHam := hamBag[tok]; !inHam {
			keep(tok, 0, s)
		}
	}

	m := &Model{
		TokenToID:    tokenToID,
		HamCounts:    hamCounts,
		SpamCounts:   spamCounts,
		hamTokens:    hamTokens,
		spamTokens:   spamTokens,
		vocabSize:    uint32(len(hamCounts)),
		logProbHam:   make([]float64, len(hamCounts)),
		logProbSpam:  make([]float64, len(hamCounts)),
		logPriorHam:  math.Log(float64(numHamDocs) / totalDocs),
		logPriorSpam: math.Log(float64(numSpamDocs) / totalDocs),
	}
	for tid := range hamCounts {
		m.logProbHam[tid] = laplaceLogProb(hamCounts[tid], hamTokens, m.vocabSize)
		m.logProbSpam[tid] = laplaceLogProb(spamCounts[tid], spamTokens, m.vocabSize)
	}
	return m, nil
}

// laplaceLogProb returns log P(token | class) with Laplace smoothing:
// log((count + 1) / (classTokens + |V|)).
func laplaceLogProb(count uint32, classTokens uint64, vocabSize uint32) float64 {
	return math.Log(float64(count+1) / float64(classTokens+uint64(vocabSize)))
}

type ClassifyResult struct {
	Label       string
	LogPostHam  float64
	LogPostSpam float64
}

// Naive Bayes on raw email bytes in log-space:
// log P( class | doc ) = log P( class ) + ( Sum for tok of log P( tok | class )) for every token in the document
func (m *Model) Classify(content []byte) ClassifyResult {
	tokens := cleanup(content)

	logHam := m.logPriorHam
	logSpam := m.logPriorSpam

	// Tokens outside the kept vocabulary (unseen or below minCount) contribute nothing.
	for _, tok := range tokens {
		if tid, ok := m.TokenToID[tok]; ok {
			logHam += m.logProbHam[tid]
			logSpam += m.logProbSpam[tid]
		}
	}

	label := "ham"
	if logSpam > logHam {
		label = "spam"
	}
	return ClassifyResult{
		Label:       label,
		LogPostHam:  logHam,
		LogPostSpam: logSpam,
	}
}
