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
)

// \p{L} matches any letter in any script (Latin, Cyrillic, Arabic, CJK, etc.) and \p{N} matches digits in any script.
var nonWordRegex = regexp.MustCompile(`[^\p{L}\p{N}\s]+`)

func cleanup(s string) []string {
	text := nonWordRegex.ReplaceAllString(strings.ToLower(s), " ")
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
		bagOfWords[token] += 1
	}
	mu.Unlock()
	return nil
}

func main() {
	freqs := map[string]int{}
	var mu sync.Mutex

	const path = "./data/enron1"
	const maxWorkers = 8

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxWorkers)

	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, walkErr error) error {
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

			if procErr := populateBagOfWords(filePath, freqs, &mu); procErr != nil {
				log.Printf("Failed to process file %s: %v", filePath, procErr)
			}
		}(p)

		return nil
	})

	if err != nil {
		log.Printf("WalkDir finished with error: %v", err)
	}

	wg.Wait()

	totalCount := 0
	for _, count := range freqs {
		totalCount += count
	}
	for token, count := range freqs {
		fmt.Printf("<%s> => %d , %f .\n", token, count, float64(count)/float64(totalCount))
	}
	fmt.Printf("Total Count is : %d\n", totalCount)
}
