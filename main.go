package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// \p{L} matches any letter in any script (Latin, Cyrillic, Arabic, CJK, etc.) and \p{N} matches digits in any script.
var nonWordRegex = regexp.MustCompile(`[^\p{L}\p{N}\s]+`)

func populateBagOfWords(path string, bagOfWords map[string]int) error {

	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("An error occurred: %v\n", err)
		return err
	}
	text := nonWordRegex.ReplaceAllString(strings.ToLower(string(content)), " ")
	tokens := strings.Fields(text)
	filtered := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if len(t) > 2 && len(t) <= 30 {
			filtered = append(filtered, t)
		}
	}
	for _, token := range filtered {
		bagOfWords[token] += 1
	}
	return nil
}

func main() {
	freqs := map[string]int{}

	const path = "./data/enron1"
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		err = populateBagOfWords(path, freqs)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		panic(err)
	}

	totalCount := 0
	for token := range freqs {
		totalCount += freqs[token]
	}
	for token := range freqs {
		fmt.Printf("<%s> => %d , %f .\n", token, freqs[token], float64(freqs[token])/float64(totalCount))
	}
	fmt.Printf("Total Count is : %d\n", totalCount)
}
