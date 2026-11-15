package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestCleanup(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"lowercases", "FrEe MONEY", []string{"free", "money"}},
		{"punctuation splits tokens", "win$$$cash!!now", []string{"win", "cash", "now"}},
		{"drops tokens under 3 runes", "a ab abc", []string{"abc"}},
		// "ça" is 2 runes but 3 bytes: the length rule counts runes.
		{"short non-ascii token dropped", "ça été", []string{"été"}},
		{"nfkc folds full-width letters", "ｆｒｅｅ", []string{"free"}},
		{"digits are kept", "call 0800 now", []string{"call", "0800", "now"}},
		{"final token captured", "hello world", []string{"hello", "world"}},
		{"empty input", "", []string{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := cleanup([]byte(tc.in)); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("cleanup(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestCleanupMaxLengthInRunes(t *testing.T) {
	ascii30 := strings.Repeat("a", 30)
	accented30 := strings.Repeat("é", 30) // 30 runes, 60 bytes
	got := cleanup([]byte(ascii30 + " " + strings.Repeat("a", 31) + " " + accented30))
	want := []string{ascii30, accented30}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}
