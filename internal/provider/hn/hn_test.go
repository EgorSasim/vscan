package hn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v0/jobstories.json":
			_, _ = w.Write([]byte(`[7,8]`))
		case "/v0/item/7.json":
			_, _ = w.Write([]byte(`{"id":7,"type":"job","title":"Acme is hiring an Angular developer","url":"https://example.com/jobs/angular","time":1790274558}`))
		case "/v0/item/8.json":
			_, _ = w.Write([]byte(`{"id":8,"type":"job","dead":true,"title":"Closed role","time":1790274558}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	p := &Provider{HTTP: &httpx.Client{HTTP: srv.Client()}, URL: srv.URL + "/v0/jobstories.json"}
	var got []provider.Vacancy
	if err := p.Search(context.Background(), nil, 0, func(l provider.Listing) {
		got = append(got, l.Vacancy)
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("%#v", got)
	}
	v := got[0]
	if v.Source != "hn" || v.URL != "https://example.com/jobs/angular" || v.Title == "" || v.Posted.IsZero() {
		t.Fatalf("%#v", v)
	}
}

func TestTextOnlyItem(t *testing.T) {
	job, ok, err := parseItem([]byte(`{"id":3,"type":"job","title":"Hiring","text":"<p>TypeScript</p>","time":1790274558}`))
	if err != nil || !ok {
		t.Fatal(err, ok)
	}
	if job.URL != "https://news.ycombinator.com/item?id=3" || job.Description != "TypeScript" {
		t.Fatalf("%#v", job)
	}
}
