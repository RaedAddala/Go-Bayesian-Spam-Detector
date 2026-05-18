package main

import "math"

type Model struct {
	TokenToID  map[string]uint32
	HamCounts  []uint32
	SpamCounts []uint32

	hamTokens  uint32
	spamTokens uint32
	vocabSize  uint32 // |V| for Laplace smoothing denominator

	minCount uint32 // tokens with hamCount+spamCount below this are skipped

	logPriorHam  float64
	logPriorSpam float64
}

// numHamDocs / numSpamDocs are document counts (emails).
func NewModel(hamBag, spamBag Bag, numHamDocs, numSpamDocs, minCount int) *Model {
	// Build a single token->ID mapping across both corpora.
	tokenToID := make(map[string]uint32, len(hamBag)+len(spamBag))

	var id uint32
	add := func(tok string) uint32 {
		if existing, ok := tokenToID[tok]; ok {
			return existing
		}
		tokenToID[tok] = id
		id++
		return id - 1
	}

	for tok := range hamBag {
		add(tok)
	}
	for tok := range spamBag {
		add(tok)
	}

	hamCounts := make([]uint32, id)
	spamCounts := make([]uint32, id)

	var hamTokens, spamTokens uint32
	for tok, c := range hamBag {
		tid := tokenToID[tok]
		uc := uint32(c)
		hamCounts[tid] = uc
		hamTokens += uc
	}
	for tok, c := range spamBag {
		tid := tokenToID[tok]
		uc := uint32(c)
		spamCounts[tid] = uc
		spamTokens += uc
	}

	totalDocs := numHamDocs + numSpamDocs
	return &Model{
		TokenToID:    tokenToID,
		HamCounts:    hamCounts,
		SpamCounts:   spamCounts,
		hamTokens:    hamTokens,
		spamTokens:   spamTokens,
		vocabSize:    uint32(id),
		minCount:     uint32(minCount),
		logPriorHam:  math.Log(float64(numHamDocs) / float64(totalDocs)),
		logPriorSpam: math.Log(float64(numSpamDocs) / float64(totalDocs)),
	}
}

// Returns log P(word | class) with Laplace smoothing.
// Returns 0 if the word's combined corpus frequency is below minCount.
func (m *Model) logWordProb(word string, spam bool) float64 {
	tid, ok := m.TokenToID[word]
	if !ok {
		return 0
	}

	hamCount := m.HamCounts[tid]
	spamCount := m.SpamCounts[tid]

	// Skip tokens too rare to be reliable signal.
	if hamCount+spamCount < m.minCount {
		return 0
	}

	var count, total uint32
	if spam {
		count, total = spamCount, m.spamTokens
	} else {
		count, total = hamCount, m.hamTokens
	}

	return math.Log(float64(count+1) / float64(total+m.vocabSize))
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

	for _, tok := range tokens {
		logHam += m.logWordProb(tok, false)
		logSpam += m.logWordProb(tok, true)
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
