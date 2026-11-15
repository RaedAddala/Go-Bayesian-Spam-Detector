package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Bag map[string]int

// Corpus holds the token counts and document counts for one or more Enron folders.
type Corpus struct {
	HamBag  Bag
	SpamBag Bag
	NumHam  int
	NumSpam int
}

// listFiles returns the paths of the regular files directly inside dir.
func listFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Type().IsRegular() {
			paths = append(paths, filepath.Join(dir, e.Name()))
		}
	}
	return paths, nil
}

// ingestDirectory tokenizes every file in dir with up to maxWorkers goroutines.
// Each worker counts into its own Bag (no shared lock); the bags are merged at the end.
// It returns the merged bag and the number of documents read, or an error if the
// directory or any file in it cannot be read.
func ingestDirectory(dir string, maxWorkers int) (Bag, int, error) {
	paths, err := listFiles(dir)
	if err != nil {
		return nil, 0, err
	}

	jobs := make(chan string)
	bags := make([]Bag, maxWorkers)
	errs := make([]error, maxWorkers)

	var wg sync.WaitGroup
	for w := range maxWorkers {
		wg.Go(func() {
			bag := make(Bag)
			for p := range jobs {
				if errs[w] != nil {
					continue // keep draining so the sender never blocks
				}
				content, err := os.ReadFile(p)
				if err != nil {
					errs[w] = err
					continue
				}
				for _, tok := range cleanup(content) {
					bag[tok]++
				}
			}
			bags[w] = bag
		})
	}
	for _, p := range paths {
		jobs <- p
	}
	close(jobs)
	wg.Wait()

	if err := errors.Join(errs...); err != nil {
		return nil, 0, err
	}

	merged := bags[0]
	for _, b := range bags[1:] {
		for tok, c := range b {
			merged[tok] += c
		}
	}
	return merged, len(paths), nil
}

// ingestFolder tokenizes <baseDir>/<folder>/ham and <baseDir>/<folder>/spam.
func ingestFolder(baseDir, folder string, maxWorkers int) (Corpus, error) {
	hamBag, numHam, err := ingestDirectory(filepath.Join(baseDir, folder, "ham"), maxWorkers)
	if err != nil {
		return Corpus{}, fmt.Errorf("%s/ham: %w", folder, err)
	}
	spamBag, numSpam, err := ingestDirectory(filepath.Join(baseDir, folder, "spam"), maxWorkers)
	if err != nil {
		return Corpus{}, fmt.Errorf("%s/spam: %w", folder, err)
	}
	return Corpus{HamBag: hamBag, SpamBag: spamBag, NumHam: numHam, NumSpam: numSpam}, nil
}

// mergeCorpora sums the bags and document counts of parts into a new Corpus.
// The inputs are not modified, so per-folder corpora can be reused across folds.
func mergeCorpora(parts ...Corpus) Corpus {
	out := Corpus{HamBag: make(Bag), SpamBag: make(Bag)}
	for _, p := range parts {
		for tok, c := range p.HamBag {
			out.HamBag[tok] += c
		}
		for tok, c := range p.SpamBag {
			out.SpamBag[tok] += c
		}
		out.NumHam += p.NumHam
		out.NumSpam += p.NumSpam
	}
	return out
}
