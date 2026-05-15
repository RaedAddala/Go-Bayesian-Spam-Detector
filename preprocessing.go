package main

import (
	"strings"
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
