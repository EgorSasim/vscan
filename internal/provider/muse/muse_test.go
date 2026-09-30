package muse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestParse(t *testing.T) {
	const body = `{"page":1,"page_count":1,"results":[{"name":"Senior Angular Developer","contents":"<p>TypeScript</p>","publication_date":"2026-09-01T00:02:34Z","company":{"name":"Acme"},"refs":{"landing_page":"https://www.themuse.com/jobs/acme/senior-angular"},"locations":[{"name":"Remote"}],"levels":[{"name":"Senior Level"}],"categories":[{"name":"Software Engineering"}]}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") != "1" {
			t.Errorf("page %s", r.URL.Query().Get("page"))
		}
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	p := &Provider{HTTP: &httpx.Client{HTTP: srv.Client()}, URL: srv.URL}
	var got []provider.Vacancy
	if err := p.Search(context.Background(), nil, 1, func(l provider.Listing) {
		got = append(got, l.Vacancy)
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("%#v", got)
	}
	v := got[0]
	if v.URL != "https://www.themuse.com/jobs/acme/senior-angular" || v.Company != "Acme" || v.Location != "Remote" {
		t.Fatalf("%#v", v)
	}
	if v.Description != "TypeScript" || v.Posted.IsZero() {
		t.Fatalf("%#v", v)
	}
}
