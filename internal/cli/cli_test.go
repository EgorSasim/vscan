package cli

import (
	"strings"
	"testing"
	"time"
)

func TestQueryAndDefaults(t *testing.T) {
	opt, err := Parse([]string{"Senior&|Angular"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.Query != "Senior&|Angular" || opt.Every != time.Hour || opt.Scanner {
		t.Fatalf("%#v", opt)
	}
}

func TestListenDoesNotEatQuery(t *testing.T) {
	opt, err := Parse([]string{"--scanner", "--listen", "Senior&Angular"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.Listen != "127.0.0.1:8787" || opt.Query != "Senior&Angular" {
		t.Fatalf("%#v", opt)
	}
	opt, err = Parse([]string{"--scanner", "--listen", "127.0.0.1:9000", "Go&Dev"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.Listen != "127.0.0.1:9000" || opt.Query != "Go&Dev" {
		t.Fatalf("%#v", opt)
	}
}

func TestHistoryNumber(t *testing.T) {
	opt, err := Parse([]string{"--history"})
	if err != nil || !opt.History || opt.HistoryN != 0 {
		t.Fatalf("%#v %v", opt, err)
	}
	opt, err = Parse([]string{"--history", "3", "--scanner"})
	if err != nil || opt.HistoryN != 3 || !opt.Scanner {
		t.Fatalf("%#v %v", opt, err)
	}
}

func TestRejectPublicListen(t *testing.T) {
	_, err := Parse([]string{"--scanner", "--listen", "0.0.0.0:8787"})
	if err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatal(err)
	}
}

func TestEveryRequiresScanner(t *testing.T) {
	_, err := Parse([]string{"--every", "15", "Go"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHelpStops(t *testing.T) {
	opt, err := Parse([]string{"--help", "--nope"})
	if err != nil || !opt.Help {
		t.Fatalf("%#v %v", opt, err)
	}
	if !strings.Contains(Help(), "--scanner") || !strings.Contains(Help(), "Senior&|Angular") {
		t.Fatal("help is missing the contract")
	}
}
