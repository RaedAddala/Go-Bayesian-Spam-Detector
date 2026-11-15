package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type labeledFile struct {
	path  string
	class string // "ham" or "spam"
}

// evaluateModel classifies every file in the ham/ and spam/ subdirs of testDir
// with up to maxWorkers goroutines and returns the summed metrics.
// The model is only read, so it is safe to share between workers.
// It returns an error if a directory or any file cannot be read.
func evaluateModel(model *Model, testDir string, maxWorkers int) (Metrics, error) {
	var files []labeledFile
	for _, class := range []string{"ham", "spam"} {
		classDir := filepath.Join(testDir, class)
		paths, err := listFiles(classDir)
		if err != nil {
			return Metrics{}, fmt.Errorf("listing %s: %w", classDir, err)
		}
		for _, p := range paths {
			files = append(files, labeledFile{path: p, class: class})
		}
	}

	jobs := make(chan labeledFile)
	partial := make([]Metrics, maxWorkers)
	errs := make([]error, maxWorkers)

	var wg sync.WaitGroup
	for w := range maxWorkers {
		wg.Go(func() {
			var met Metrics
			for f := range jobs {
				if errs[w] != nil {
					continue // keep draining so the sender never blocks
				}
				content, err := os.ReadFile(f.path)
				if err != nil {
					errs[w] = err
					continue
				}
				met.record(f.class, model.Classify(content).Label)
			}
			partial[w] = met
		})
	}
	for _, f := range files {
		jobs <- f
	}
	close(jobs)
	wg.Wait()

	if err := errors.Join(errs...); err != nil {
		return Metrics{}, err
	}

	var total Metrics
	for _, m := range partial {
		total = total.Add(m)
	}
	return total, nil
}
