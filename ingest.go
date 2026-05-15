package main

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type Bag map[string]int

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
