package geekjob

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestJSONAndDescription(t *testing.T) {
	const list = `{"page":1,"nextpage":0,"documentsCount":1,"data":[{"id":"abc","position":"Senior Golang Developer","company":{"name":"Cryptopay"},"jobFormat":{"remote":true,"relocate":false}}]}`
	const page = `<html><div class="vacancy-description"><p>Payments platform. Relocation is possible.</p></div></html>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/vacancy/") {
			_, _ = w.Write([]byte(page))
			return
		}
		_, _ = w.Write([]byte(list))
	}))
	defer srv.Close()
	p := &Provider{HTTP: &httpx.Client{HTTP: srv.Client()}, Base: srv.URL}
	var got []provider.Listing
	if err := p.Search(context.Background(), []string{"golang"}, 1, func(l provider.Listing) {
		got = append(got, l)
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Vacancy.Tags[0] != "remote" || got[0].Vacancy.Company != "Cryptopay" {
		t.Fatalf("%#v", got)
	}
	full, err := got[0].Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(full.Description, "Relocation") {
		t.Fatalf("%#v", full)
	}
}
