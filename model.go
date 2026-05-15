package main

import "math"

type Model struct {
	hamBag  Bag
	spamBag Bag

	hamTokens  int
	spamTokens int
	vocabSize  int // |V| for Laplace smoothing denominator

	minCount int // tokens with hamCount+spamCount below this are skipped

	logPriorHam  float64
	logPriorSpam float64
}

// numHamDocs / numSpamDocs are document counts (emails).
func NewModel(hamBag, spamBag Bag, numHamDocs, numSpamDocs, minCount int) *Model {
	// Compute vocab size
	vocabSize := len(hamBag)
	for w := range spamBag {
		if _, inHam := hamBag[w]; !inHam {
			vocabSize++
		}
	}

	total := numHamDocs + numSpamDocs
	return &Model{
		hamBag:       hamBag,
		spamBag:      spamBag,
		hamTokens:    countBag(hamBag),
		spamTokens:   countBag(spamBag),
		vocabSize:    vocabSize,
		minCount:     minCount,
		logPriorHam:  math.Log(float64(numHamDocs) / float64(total)),
		logPriorSpam: math.Log(float64(numSpamDocs) / float64(total)),
	}
}

// Returns log P(word | class) with Laplace smoothing.
// Returns 0 if the word's combined corpus frequency is below minCount,
func (m *Model) logWordProb(word string, spam bool) float64 {
	hamCount := m.hamBag[word]
	spamCount := m.spamBag[word]

	// Skip tokens too rare to be reliable signal.
	if hamCount+spamCount < m.minCount {
		return 0
	}

	var count, total int
	if spam {
		count, total = spamCount, m.spamTokens
	} else {
		count, total = hamCount, m.hamTokens
	}
	// IMPORTANT!! Note so it doesn't confuse future me
	// this is Laplace Smoothing: adding 1 in the numerator to avoid log(0) which gives -Inf
	// In this case if we add 1 to every word's count in the numerator then we inflate the total count by exactly vocabSize
	// We add vocabSize to balance this out
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
