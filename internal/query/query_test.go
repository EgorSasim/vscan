package query

import (
	"os"
	"path/filepath"
	"testing"
)

func mustParse(t *testing.T, q string) Expr {
	t.Helper()
	e, err := Parse(q)
	if err != nil {
		t.Fatalf("parse %q: %v", q, err)
	}
	return e
}

func TestSmartRemoteAcrossFields(t *testing.T) {
	e := mustParse(t, "Senior&|Angular&|Remote")
	text := "Senior Angular Developer\n\nWe help people relocate to the EU."
	if !Match(e, text, nil) {
		t.Fatal("title Senior Angular and description relocate must match")
	}
}

func TestLiteralDoesNotUseSynonymOrPrefix(t *testing.T) {
	e := mustParse(t, "Senior&Angular")
	if Match(e, "Senior AngularJS Developer relocate", nil) {
		t.Fatal("literal Angular must not match AngularJS")
	}
	if Match(e, "Sr Angular developer", nil) {
		t.Fatal("literal Senior must not match Sr")
	}
	if !Match(e, "We need a Senior Angular developer", nil) {
		t.Fatal("both whole words in one text must match")
	}
}

func TestMixedChainModes(t *testing.T) {
	e := mustParse(t, "Senior&Angular&|Remote")
	if !Match(e, "Senior Angular Developer. Relocation package.", nil) {
		t.Fatal("Remote is smart and must accept relocate")
	}
	if Match(e, "Sr Angular Developer remote", nil) {
		t.Fatal("Senior stays literal in a mixed chain")
	}
	smartFirst := mustParse(t, "Senior&|Angular&Remote")
	if Match(smartFirst, "Senior Angular Developer relocate", nil) {
		t.Fatal("last term Remote is literal and must not accept relocate")
	}
	if !Match(smartFirst, "Sr. Angular role, remote team", nil) {
		t.Fatal("Senior and Angular are smart, Remote is the literal word")
	}
}

func TestOrPrecedence(t *testing.T) {
	e := mustParse(t, "Senior&Angular||React")
	if !Match(e, "React developer", nil) {
		t.Fatal("OR branch React")
	}
	if !Match(e, "Senior Angular developer", nil) {
		t.Fatal("AND branch")
	}
	if Match(e, "Senior Go developer", nil) {
		t.Fatal("Senior alone is not enough")
	}
	got := Hints(e)
	if len(got) != 2 || got[0] != "Senior Angular" || got[1] != "React" {
		t.Fatalf("hints = %#v", got)
	}
}

func TestParentheses(t *testing.T) {
	e := mustParse(t, "(Senior||Lead)&Angular")
	if !Match(e, "Lead Angular developer", nil) {
		t.Fatal("Lead Angular")
	}
	if Match(e, "Lead Go developer", nil) {
		t.Fatal("Angular is required")
	}
	hints := Hints(e)
	if len(hints) != 2 || hints[0] != "Senior Angular" || hints[1] != "Lead Angular" {
		t.Fatalf("hints = %#v", hints)
	}
}

func TestQuotedPhrase(t *testing.T) {
	e := mustParse(t, `Senior&"remote work"`)
	if !Match(e, "Senior engineer, remote work is ok", nil) {
		t.Fatal("quoted phrase")
	}
	if Match(e, "Senior engineer remote", nil) {
		t.Fatal("phrase must stay together")
	}
}

func TestYoAndCase(t *testing.T) {
	e := mustParse(t, "Удалённо&|Angular")
	if !Match(e, "angular, удаленно", nil) {
		t.Fatal("ё and case")
	}
}

func TestPluralAndPrefixSmart(t *testing.T) {
	e := mustParse(t, "Angular&|Developer")
	if !Match(e, "AngularJS developers wanted", nil) {
		t.Fatal("prefix and plural")
	}
}

func TestShortSmartWordIsWhole(t *testing.T) {
	e := mustParse(t, "Go&|Developer")
	if Match(e, "Google developer", nil) {
		t.Fatal("two-letter Go must not prefix-match Google")
	}
	if !Match(e, "Go developer", nil) {
		t.Fatal("whole word Go")
	}
}

func TestRejectBareTermsAndBadOps(t *testing.T) {
	if _, err := Parse("Senior Angular"); err == nil {
		t.Fatal("expected error without an operator")
	}
	if _, err := Parse("Senior|Angular"); err == nil {
		t.Fatal("expected error on single bar")
	}
	if _, err := Parse(""); err == nil {
		t.Fatal("expected empty error")
	}
}

func TestSynonymFileOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "synonyms")
	body := "remote = distributed\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	syn := BuiltinSynonyms()
	if err := syn.MergeFile(path); err != nil {
		t.Fatal(err)
	}
	e := mustParse(t, "Remote&|Angular")
	if Match(e, "Angular relocate", syn) {
		t.Fatal("override must drop relocate from the remote group")
	}
	if !Match(e, "Angular distributed team", syn) {
		t.Fatal("override alias distributed")
	}
	if err := syn.MergeFile(filepath.Join(dir, "missing")); err != nil {
		t.Fatal(err)
	}
}
