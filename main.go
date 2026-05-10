package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/text/unicode/norm"
)

// \p{L} matches any letter in any script (Latin, Cyrillic, Arabic, CJK, etc.) and \p{N} matches digits in any script.
var nonWordRegex = regexp.MustCompile(`[^\p{L}\p{N}\s]+`)

func cleanup(s string) []string {
	normalized := norm.NFKC.String(s)
	text := nonWordRegex.ReplaceAllString(strings.ToLower(normalized), " ")
	tokens := strings.Fields(text)
	filtered := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if len(t) > 2 && len(t) <= 30 {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func populateBagOfWords(path string, bagOfWords map[string]int, mu *sync.Mutex) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	filtered := cleanup(string(content))
	mu.Lock()
	for _, token := range filtered {
		bagOfWords[token]++
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
