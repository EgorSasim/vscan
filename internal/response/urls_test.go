package response

import "testing"

func TestParseLinksSeparators(t *testing.T) {
	text := "https://hh.ru/vacancy/1\x1fhttps://career.habr.com/vacancies/2\nhttps://remoteok.com/remote-jobs/3"
	got := ParseLinks(text)
	if len(got) != 3 {
		t.Fatalf("len %d", len(got))
	}
	if !got[0].Apply || got[0].Board != "hh" {
		t.Fatalf("hh %#v", got[0])
	}
	if !got[1].Apply || got[1].Board != "habr" {
		t.Fatalf("habr %#v", got[1])
	}
	if got[2].Apply || got[2].Problem == "" {
		t.Fatalf("aggregator should be skipped: %#v", got[2])
	}
}

func TestParseLinksRejectsJunk(t *testing.T) {
	got := ParseLinks("not a url\nhttps://example.com/jobs/1")
	if len(got) != 2 || got[0].Problem == "" || got[1].Problem == "" {
		t.Fatalf("%#v", got)
	}
}
