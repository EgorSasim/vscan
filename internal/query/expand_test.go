package query

import "testing"

func TestExpandAlias(t *testing.T) {
	aliases := map[string]string{
		"myStack": "Angular||Typescript||JavaScript||JS||TS",
	}
	got, err := Expand("Senior&myStack", aliases)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Senior&(Angular||Typescript||JavaScript||JS||TS)" {
		t.Fatalf("got %q", got)
	}
	e, err := Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if !Match(e, "Senior TS developer", nil) {
		t.Fatal("expanded alias must match TS")
	}
	if Match(e, "Senior Go developer", nil) {
		t.Fatal("Go is outside the alias")
	}
}

func TestExpandQuotedIsLiteral(t *testing.T) {
	got, err := Expand(`"myStack"`, map[string]string{"myStack": "Angular"})
	if err != nil {
		t.Fatal(err)
	}
	if got != `"myStack"` {
		t.Fatalf("got %q", got)
	}
}

func TestExpandCycle(t *testing.T) {
	_, err := Expand("a", map[string]string{"a": "b", "b": "a"})
	if err == nil {
		t.Fatal("expected cycle error")
	}
}
