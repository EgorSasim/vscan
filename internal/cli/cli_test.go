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

func TestFieldFlagsAndAlias(t *testing.T) {
	opt, err := Parse([]string{"--company=TBank|AlfaBank", "--profession=myStack", "--platform=hh|habr"})
	if err != nil {
		t.Fatal(err)
	}
	if opt.Company != "TBank|AlfaBank" || opt.Profession != "myStack" || opt.Platform != "hh|habr" || opt.Query != "" {
		t.Fatalf("%#v", opt)
	}
	rec := opt.SearchRecord()
	args, err := Split(rec)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Parse(args)
	if err != nil {
		t.Fatal(err)
	}
	if again.Company != opt.Company || again.Profession != opt.Profession || again.Platform != opt.Platform {
		t.Fatalf("roundtrip %#v from %q", again, rec)
	}
	set, err := Parse([]string{"alias", "myStack=Angular|TS"})
	if err != nil || set.AliasName != "myStack" || set.AliasValue != "Angular|TS" {
		t.Fatalf("%#v %v", set, err)
	}
	list, err := Parse([]string{"alias"})
	if err != nil || !list.AliasList {
		t.Fatalf("%#v %v", list, err)
	}
	del, err := Parse([]string{"alias", "--delete", "myStack"})
	if err != nil || del.AliasDel != "myStack" {
		t.Fatalf("%#v %v", del, err)
	}
}

func TestHelpStops(t *testing.T) {
	opt, err := Parse([]string{"--help", "--nope"})
	if err != nil || !opt.Help {
		t.Fatalf("%#v %v", opt, err)
	}
	if !strings.Contains(Help(), "--scanner") || !strings.Contains(Help(), "Senior&|Angular") || !strings.Contains(Help(), "--verbose") {
		t.Fatal("help is missing the contract")
	}
}

func TestResponseAndCredsFlags(t *testing.T) {
	opt, err := Parse([]string{"--response", "https://hh.ru/vacancy/1"})
	if err != nil || !opt.Response || opt.ResponseArg == "" {
		t.Fatalf("%#v %v", opt, err)
	}
	opt, err = Parse([]string{"--headed", "--response"})
	if err != nil || !opt.Headed || !opt.Response {
		t.Fatalf("%#v %v", opt, err)
	}
	if _, err := Parse([]string{"--headed", "Go"}); err == nil {
		t.Fatal("--headed without --response")
	}
	opt, err = Parse([]string{"creds", "set", "hh"})
	if err != nil || opt.CredsAction != "set" || opt.CredsPlatform != "hh" {
		t.Fatalf("%#v %v", opt, err)
	}
}

func TestPresetsCommand(t *testing.T) {
	opt, err := Parse([]string{"presets"})
	if err != nil || !opt.Presets {
		t.Fatalf("%#v %v", opt, err)
	}
	if _, err := Parse([]string{"presets", "Go"}); err == nil {
		t.Fatal("presets takes no arguments")
	}
}

func TestScanVerboseAndClearCache(t *testing.T) {
	opt, err := Parse([]string{"--scan", "--verbose", "--clear-cache", "--every", "1", "Go"})
	if err != nil || !opt.Scanner || !opt.Verbose || !opt.ClearCache || opt.Query != "Go" {
		t.Fatalf("%#v %v", opt, err)
	}
}
