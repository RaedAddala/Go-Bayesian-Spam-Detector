package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeFiles(t *testing.T, dir string, contents ...string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for i, c := range contents {
		if err := os.WriteFile(filepath.Join(dir, string(rune('a'+i))+".txt"), []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestIngestFolder(t *testing.T) {
	base := t.TempDir()
	writeFiles(t, filepath.Join(base, "f1", "ham"), "hello meeting", "meeting notes")
	writeFiles(t, filepath.Join(base, "f1", "spam"), "free money")

	c, err := ingestFolder(base, "f1", 3)
	if err != nil {
		t.Fatal(err)
	}
	if c.NumHam != 2 || c.NumSpam != 1 {
		t.Errorf("NumHam=%d NumSpam=%d, want 2 and 1", c.NumHam, c.NumSpam)
	}
	if want := (Bag{"hello": 1, "meeting": 2, "notes": 1}); !reflect.DeepEqual(c.HamBag, want) {
		t.Errorf("HamBag = %v, want %v", c.HamBag, want)
	}
}

func TestIngestFolderMissingDirIsAnError(t *testing.T) {
	base := t.TempDir()
	writeFiles(t, filepath.Join(base, "f1", "ham"), "hello")
	if _, err := ingestFolder(base, "f1", 2); err == nil {
		t.Error("expected an error for a missing spam directory")
	}
}

func TestMergeCorporaDoesNotMutateInputs(t *testing.T) {
	a := Corpus{HamBag: Bag{"abc": 1}, SpamBag: Bag{"xyz": 2}, NumHam: 1, NumSpam: 1}
	b := Corpus{HamBag: Bag{"abc": 3}, SpamBag: Bag{}, NumHam: 2, NumSpam: 0}

	got := mergeCorpora(a, b)
	if got.HamBag["abc"] != 4 || got.SpamBag["xyz"] != 2 || got.NumHam != 3 || got.NumSpam != 1 {
		t.Errorf("unexpected merge result %+v", got)
	}
	if a.HamBag["abc"] != 1 || b.HamBag["abc"] != 3 {
		t.Error("mergeCorpora modified its inputs")
	}
}

func TestEvaluateModel(t *testing.T) {
	m, err := NewModel(Bag{"meeting": 10}, Bag{"viagra": 10}, 1, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writeFiles(t, filepath.Join(dir, "ham"), "meeting", "viagra")
	writeFiles(t, filepath.Join(dir, "spam"), "viagra viagra")

	met, err := evaluateModel(m, dir, 4)
	if err != nil {
		t.Fatal(err)
	}
	want := Metrics{Total: 3, Correct: 2, TP: 1, FP: 1, TN: 1, FN: 0}
	if met != want {
		t.Errorf("metrics = %+v, want %+v", met, want)
	}
}
