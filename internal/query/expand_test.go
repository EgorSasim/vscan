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

func TestPresetsExpand(t *testing.T) {
	got, err := ExpandPresets("Go", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "(Golang||Go)" {
		t.Fatalf("got %q", got)
	}
	got, err = ExpandPresets("angular", nil)
	if err != nil || got != "(Angular||AngularJS)" {
		t.Fatalf("got %q %v", got, err)
	}
	got, err = ExpandPresets(`"Go"`, nil)
	if err != nil || got != `"Go"` {
		t.Fatalf("quoted got %q %v", got, err)
	}
	got, err = ExpandPresets("Senior&Go", nil)
	if err != nil || got != "Senior&(Golang||Go)" {
		t.Fatalf("mixed got %q %v", got, err)
	}
	got, err = ExpandPresets("Go", map[string]string{"go": "OnlyThisWord"})
	if err != nil || got != "(OnlyThisWord)" {
		t.Fatalf("alias must win, got %q %v", got, err)
	}

	e, err := Parse(mustExpand(t, "Go"))
	if err != nil {
		t.Fatal(err)
	}
	if !Match(e, "Senior Golang developer", nil) || !Match(e, "Go developer", nil) {
		t.Fatal("Go preset must match Golang and Go")
	}
	if Match(e, "Google developer", nil) {
		t.Fatal("Go preset must not match Google")
	}
	java, err := Parse(mustExpand(t, "Java"))
	if err != nil {
		t.Fatal(err)
	}
	if Match(java, "JavaScript developer", nil) {
		t.Fatal("Java preset must not match JavaScript")
	}
	if !Match(java, "Java developer", nil) {
		t.Fatal("Java preset")
	}
	js, err := Parse(mustExpand(t, "JS"))
	if err != nil {
		t.Fatal(err)
	}
	if !Match(js, "JavaScript developer", nil) {
		t.Fatal("JS preset")
	}
}

func mustExpand(t *testing.T, q string) string {
	t.Helper()
	got, err := ExpandPresets(q, nil)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestExpandCycle(t *testing.T) {
	_, err := Expand("a", map[string]string{"a": "b", "b": "a"})
	if err == nil {
		t.Fatal("expected cycle error")
	}
}
