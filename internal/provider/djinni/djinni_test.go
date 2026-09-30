package djinni

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestListAndDetail(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/jobs/123") {
			_, _ = w.Write([]byte(`<a href="/jobs/123-senior-golang">Senior Golang</a><div class="job-post__description"><p>office in Berlin, relocate</p></div>`))
			return
		}
		_, _ = w.Write([]byte(`<a href="/jobs/123-senior-golang">Senior Golang</a>`))
	}))
	defer srv.Close()
	p := &Provider{HTTP: &httpx.Client{HTTP: srv.Client()}, Base: srv.URL}
	var got []provider.Listing
	if err := p.Search(context.Background(), []string{"golang"}, 1, func(l provider.Listing) {
		got = append(got, l)
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Vacancy.Title != "Senior Golang" {
		t.Fatalf("%#v", got)
	}
	full, err := got[0].Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(full.Description, "relocate") {
		t.Fatalf("%#v", full)
	}
}

func TestEmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	p := &Provider{HTTP: &httpx.Client{HTTP: srv.Client()}, Base: srv.URL}
	err := p.Search(context.Background(), []string{"go"}, 1, func(provider.Listing) {})
	if err == nil || !strings.Contains(err.Error(), "пустой") {
		t.Fatalf("err = %v", err)
	}
}
