package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

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
