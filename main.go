package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// \p{L} matches any letter in any script (Latin, Cyrillic, Arabic, CJK, etc.) and \p{N} matches digits in any script.
var nonWordRegex = regexp.MustCompile(`[^\p{L}\p{N}\s]+`)

func main() {

	const filepath = "./data/enron1/ham/0002.1999-12-13.farmer.ham.txt"
	content, err := os.ReadFile(filepath)
	if err != nil {
		fmt.Printf("An error occurred: %v\n", err)
		return
	}
	text := strings.ToLower(string(content))
	text = nonWordRegex.ReplaceAllString(text, " ")
	tokens := strings.Fields(text)
	freqs := map[string]int{}
	filtered := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if len(t) > 2 && len(t) <= 30 {
			filtered = append(filtered, t)
		}
	}
	for _, token := range filtered {
		freqs[token] += 1
	}
	totalCount := 0
	for token := range freqs {
		totalCount += freqs[token]
	}
	for token := range freqs {
		fmt.Printf("<%s> => %d , %f .\n", token, freqs[token], float64(freqs[token])/float64(totalCount))
	}
}
