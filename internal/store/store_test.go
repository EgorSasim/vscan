package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHistoryPushMovesDuplicateToTop(t *testing.T) {
	d := Dirs{Data: t.TempDir()}
	h := OpenHistory(d)
	for _, q := range []string{"A&B", "C||D", "A&B"} {
		if err := h.Push(q); err != nil {
			t.Fatal(err)
		}
	}
	lines, err := h.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[0] != "A&B" || lines[1] != "C||D" {
		t.Fatalf("history = %#v", lines)
	}
	got, err := h.Get(2)
	if err != nil || got != "C||D" {
		t.Fatalf("get 2 = %q %v", got, err)
	}
	if _, err := h.Get(3); err == nil {
		t.Fatal("expected missing entry")
	}
}

func TestSeenAppendAndInitialEmpty(t *testing.T) {
	d := Dirs{Cache: t.TempDir()}
	s, err := OpenSeen(d, "Senior&Angular", []string{"habr", "hh"})
	if err != nil {
		t.Fatal(err)
	}
	if !s.InitialEmpty() {
		t.Fatal("new cache must be empty")
	}
	if err := s.Add("https://example.com/1"); err != nil {
		t.Fatal(err)
	}
	if !s.InitialEmpty() {
		t.Fatal("InitialEmpty is fixed at open")
	}
	if !s.Has("https://example.com/1") {
		t.Fatal("missing url")
	}
	again, err := OpenSeen(d, "Senior&Angular", []string{"habr", "hh"})
	if err != nil {
		t.Fatal(err)
	}
	if again.InitialEmpty() || !again.Has("https://example.com/1") {
		t.Fatal("reload")
	}
	other, err := OpenSeen(d, "Other", []string{"habr", "hh"})
	if err != nil {
		t.Fatal(err)
	}
	if other.Has("https://example.com/1") || !other.InitialEmpty() {
		t.Fatal("query hash must isolate caches")
	}
	if _, err := os.Stat(filepath.Join(d.Cache, "seen")); err != nil {
		t.Fatal(err)
	}
}
