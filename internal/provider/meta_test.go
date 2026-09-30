package provider

import (
	"testing"
	"time"
)

func TestDisplayRemote(t *testing.T) {
	if (Vacancy{Remote: "no", Description: "remote ok"}).DisplayRemote() != "no" {
		t.Fatal("explicit no wins")
	}
	if (Vacancy{Title: "Backend", Description: "можно удалённо"}).DisplayRemote() != "yes" {
		t.Fatal("description")
	}
	if (Vacancy{Tags: []string{"office"}}).DisplayRemote() != "" {
		t.Fatal("unknown stays empty")
	}
}

func TestAgeAndSalary(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	posted := now.Add(-48 * time.Hour)
	if Age(posted, now) != "2 days" {
		t.Fatal(Age(posted, now))
	}
	if Age(now, now) != "today" {
		t.Fatal(Age(now, now))
	}
	if FormatSalary(100, 200, "USD", "yearly") != "100-200 USD yearly" {
		t.Fatal(FormatSalary(100, 200, "USD", "yearly"))
	}
	got := ParseTime("2026-09-21T12:55:11")
	if got.IsZero() || got.Year() != 2026 {
		t.Fatal(got)
	}
}
