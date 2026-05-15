package main

import (
	"fmt"
	"io/fs"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

type Bag map[string]int
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

func cleanup(content []byte) []string {
	normalized := norm.NFKC.Bytes(content)

	filtered := make([]string, 0, len(normalized)/5) // Pre-allocate slice capacity to reduce memory re-allocations
	var word strings.Builder
	word.Grow(30) // Pre-size the builder for an average word

	for i := 0; i < len(normalized); {
		r, size := utf8.DecodeRune(normalized[i:])
		i += size

		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			word.WriteRune(unicode.ToLower(r))
		} else {
			if word.Len() > 2 && word.Len() <= 30 {
				filtered = append(filtered, word.String())
			}
			word.Reset()
		}
	}

	// Capture the final word
	if word.Len() > 2 && word.Len() <= 30 {
		filtered = append(filtered, word.String())
	}
	return filtered
}

func populateBagOfWords(path string, bow Bag) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	filtered := cleanup(content)

	for _, token := range filtered {
		bow[token]++
	}
	return nil
}

func populateBagOfWordsWithLock(path string, globalBag Bag, mu *sync.Mutex) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	filtered := cleanup(content)

	localBag := make(Bag)
	for _, token := range filtered {
		localBag[token]++
	}

	mu.Lock()
	for token, count := range localBag {
		globalBag[token] += count
	}
	mu.Unlock()
	return nil
}

// Process a directory (ham or spam)
func processDirectory(dirPath string, bag Bag, mu *sync.Mutex, maxWorkers int) error {
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxWorkers)

	err := filepath.WalkDir(dirPath, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			log.Printf("Walk error at %s: %v", p, walkErr)
			return nil
		}
		if d.IsDir() {
			return nil
		}

		sem <- struct{}{}
		wg.Add(1)

		go func(filePath string) {
			defer wg.Done()
			defer func() { <-sem }()

			if procErr := populateBagOfWordsWithLock(filePath, bag, mu); procErr != nil {
				log.Printf("Failed to process file %s: %v", filePath, procErr)
			}
		}(p)

		return nil
	})

	if err != nil {
		log.Printf("WalkDir error in %s: %v", dirPath, err)
	}

	wg.Wait()
	return nil
}

func printBag(label string, bag Bag) {
	total := countBag(bag)
	fmt.Printf("Total tokens in %s: %d\n", label, total)
	// Show well used tokens
	for token, count := range bag {
		if count >= 100 {
			prob := float64(count) / float64(total)
			fmt.Printf("<%s> => %d (%.6f)\n", token, count, prob)
		}
	}
}

func countBag(bag Bag) int {
	total := 0
	for _, count := range bag {
		total += count
	}
	return total
}

func countFiles(dirPath string) (int, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() {
			n++
		}
	}
	return n, nil
}

func ingestFolders(baseDir string, folders []string, maxWorkers int) (Bag, Bag, int, int, error) {
	hamBag := make(Bag)
	spamBag := make(Bag)

	var hamMu, spamMu, countMu sync.Mutex
	var numHamDocs, numSpamDocs int
	var wg sync.WaitGroup

	for _, folder := range folders {
		hamPath := filepath.Join(baseDir, folder, "ham")
		spamPath := filepath.Join(baseDir, folder, "spam")

		wg.Add(2)

		go func(p string) {
			defer wg.Done()
			if err := processDirectory(p, hamBag, &hamMu, maxWorkers); err != nil {
				log.Printf("Error processing %s: %v", p, err)
			}
			n, err := countFiles(p)
			if err != nil {
				log.Printf("Error counting %s: %v", p, err)
				return
			}
			countMu.Lock()
			numHamDocs += n
			countMu.Unlock()
		}(hamPath)

		go func(p string) {
			defer wg.Done()
			if err := processDirectory(p, spamBag, &spamMu, maxWorkers); err != nil {
				log.Printf("Error processing %s: %v", p, err)
			}
			n, err := countFiles(p)
			if err != nil {
				log.Printf("Error counting %s: %v", p, err)
				return
			}
			countMu.Lock()
			numSpamDocs += n
			countMu.Unlock()
		}(spamPath)
	}

	wg.Wait()
	return hamBag, spamBag, numHamDocs, numSpamDocs, nil
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

type Metrics struct {
	Total   int
	Correct int
	TP      int // spam predicted spam
	FP      int // ham predicted spam
	TN      int // ham predicted ham
	FN      int // spam predicted ham
}

func (met Metrics) Accuracy() float64 { return float64(met.Correct) / float64(met.Total) }
func (met Metrics) Precision() float64 {
	if met.TP+met.FP == 0 {
		return 0
	}
	return float64(met.TP) / float64(met.TP+met.FP)
}
func (met Metrics) Recall() float64 {
	if met.TP+met.FN == 0 {
		return 0
	}
	return float64(met.TP) / float64(met.TP+met.FN)
}
func (met Metrics) F1() float64 {
	p, r := met.Precision(), met.Recall()
	if p+r == 0 {
		return 0
	}
	return 2 * p * r / (p + r)
}

// walks the ham/ and spam/ subdirs of testDir
// classifies every file with the model, and returns metrics.
func evaluateModel(model *Model, testDir string) (Metrics, error) {
	var met Metrics

	for _, class := range []string{"ham", "spam"} {
		classDir := filepath.Join(testDir, class)
		err := filepath.WalkDir(classDir, func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() {
				return walkErr
			}
			content, err := os.ReadFile(p)
			if err != nil {
				log.Printf("Cannot read %s: %v", p, err)
				return nil
			}

			res := model.Classify(content)
			met.Total++
			if res.Label == class {
				met.Correct++
			}
			switch {
			case class == "spam" && res.Label == "spam":
				met.TP++
			case class == "ham" && res.Label == "spam":
				met.FP++
			case class == "ham" && res.Label == "ham":
				met.TN++
			case class == "spam" && res.Label == "ham":
				met.FN++
			}
			return nil
		})
		if err != nil {
			return met, fmt.Errorf("walking %s: %w", classDir, err)
		}
	}
	return met, nil
}

func main() {
	const (
		baseDir    = "./data"
		maxWorkers = 8
		minCount   = 20
	)

	trainFolders := []string{"enron1", "enron2", "enron3", "enron4", "enron5"}
	testFolder := filepath.Join(baseDir, "enron6")

	fmt.Println("Ingesting training folders…")
	hamBag, spamBag, numHam, numSpam, err := ingestFolders(baseDir, trainFolders, maxWorkers)
	if err != nil {
		log.Fatalf("Ingestion failed: %v", err)
	}
	fmt.Printf("Training corpus: %d ham docs, %d spam docs\n", numHam, numSpam)

	model := NewModel(hamBag, spamBag, numHam, numSpam, minCount)
	fmt.Printf("Vocab size: %d  (minCount=%d)\n", model.vocabSize, minCount)
	fmt.Printf("log-prior  ham=%.4f  spam=%.4f\n\n", model.logPriorHam, model.logPriorSpam)

	fmt.Println("Evaluating on enron6…")
	met, err := evaluateModel(model, testFolder)
	if err != nil {
		log.Fatalf("Evaluation failed: %v", err)
	}

	fmt.Printf("Results on %d emails:\n", met.Total)
	fmt.Printf("Accuracy  : %.4f\n", met.Accuracy())
	fmt.Printf("Precision : %.4f\n", met.Precision())
	fmt.Printf("Recall    : %.4f\n", met.Recall())
	fmt.Printf("F1        : %.4f\n", met.F1())
	fmt.Printf("TP=%d  FP=%d  TN=%d  FN=%d\n", met.TP, met.FP, met.TN, met.FN)
}
