package hh

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestSearchEmitsSnippetThenDetail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/vacancies/42" {
			_, _ = w.Write([]byte(`{"id":"42","name":"Senior Angular","alternate_url":"https://hh.ru/vacancy/42","description":"<p>relocation package</p>","key_skills":[{"name":"Angular"}],"employer":{"name":"Acme"}}`))
			return
		}
		if !strings.Contains(r.URL.RawQuery, "text=Senior+Angular") && !strings.Contains(r.URL.RawQuery, "text=Senior%20Angular") {
			t.Errorf("query = %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"pages":1,"items":[{"id":42,"name":"Senior Angular","alternate_url":"https://hh.ru/vacancy/42","employer":{"name":"Acme"},"snippet":{"requirement":"Angular","responsibility":"build UI"}}]}`))
	}))
	defer srv.Close()
	p := &Provider{HTTP: &httpx.Client{HTTP: srv.Client()}, Base: srv.URL}
	var got []provider.Listing
	if err := p.Search(context.Background(), []string{"Senior Angular"}, 1, func(l provider.Listing) {
		got = append(got, l)
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Vacancy.URL != "https://hh.ru/vacancy/42" {
		t.Fatalf("listings = %#v", got)
	}
	full, err := got[0].Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(full.Description, "relocation") || full.Skills[0] != "Angular" {
		t.Fatalf("detail = %#v", full)
	}
}

func TestNumericID(t *testing.T) {
	page, err := parseSearch([]byte(`{"pages":1,"items":[{"id":7,"name":"Go","alternate_url":"https://hh.ru/vacancy/7"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if page.Items[0].ID != "7" {
		t.Fatalf("id = %s", page.Items[0].ID)
	}
}
