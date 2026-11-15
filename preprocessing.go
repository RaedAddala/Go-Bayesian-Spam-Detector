package main

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Token length bounds, measured in runes (characters), not bytes.
const (
	minTokenRunes = 3
	maxTokenRunes = 30
)

func cleanup(content []byte) []string {
	normalized := norm.NFKC.Bytes(content)

	filtered := make([]string, 0, len(normalized)/5) // Pre-allocate slice capacity to reduce memory re-allocations
	var word strings.Builder
	word.Grow(30) // Pre-size the builder for an average word
	runes := 0    // rune count of the current word; word.Len() is in bytes

	flush := func() {
		if runes >= minTokenRunes && runes <= maxTokenRunes {
			filtered = append(filtered, word.String())
		}
		word.Reset()
		runes = 0
	}

	for i := 0; i < len(normalized); {
		r, size := utf8.DecodeRune(normalized[i:])
		i += size

		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			word.WriteRune(unicode.ToLower(r))
			runes++
		} else {
			flush()
		}
	}

	// Capture the final word
	flush()
	return filtered
}
