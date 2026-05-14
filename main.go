package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

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

func populateBagOfWords(path string, globalBag map[string]int, mu *sync.Mutex) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	filtered := cleanup(content)

	localBag := make(map[string]int)
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
func processDirectory(dirPath string, bag map[string]int, mu *sync.Mutex, maxWorkers int) error {
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

			if procErr := populateBagOfWords(filePath, bag, mu); procErr != nil {
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

func main() {
	const basePath = "./data/enron1"
	const maxWorkers = 8

	hamBag := make(map[string]int)
	spamBag := make(map[string]int)

	var hamMu, spamMu sync.Mutex

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		hamPath := filepath.Join(basePath, "ham")
		if err := processDirectory(hamPath, hamBag, &hamMu, maxWorkers); err != nil {
			log.Printf("Error processing ham: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		spamPath := filepath.Join(basePath, "spam")
		if err := processDirectory(spamPath, spamBag, &spamMu, maxWorkers); err != nil {
			log.Printf("Error processing spam: %v", err)
		}
	}()

	wg.Wait()

	fmt.Println("=== HAM BAG OF WORDS ===")
	printBag("ham", hamBag)

	fmt.Println("\n=== SPAM BAG OF WORDS ===")
	printBag("spam", spamBag)
}

func printBag(label string, bag map[string]int) {
	total := 0
	for _, count := range bag {
		total += count
	}

	fmt.Printf("Total tokens in %s: %d\n", label, total)
	// Show well used tokens
	for token, count := range bag {
		if count >= 100 {
			prob := float64(count) / float64(total)
			fmt.Printf("<%s> => %d (%.6f)\n", token, count, prob)
		}
	}
}
