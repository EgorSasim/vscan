package habr

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestRSSAndJSONLD(t *testing.T) {
	const page = `<html><script type="application/ld+json">{"@context":"https://schema.org/","@type":"JobPosting","title":"Senior Golang","description":"<p>relocation and PostgreSQL</p>","hiringOrganization":{"@type":"Organization","name":"Acme"}}</script></html>`
	var rss string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/vacancies/100") {
			_, _ = w.Write([]byte(page))
			return
		}
		_, _ = w.Write([]byte(rss))
	}))
	defer srv.Close()
	rss = `<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"><channel><title>t</title><item><title>Требуется «Senior Golang»</title><description>Навыки: #golang. Можно удалённо.</description><author>Acme</author><link>` + srv.URL + `/vacancies/100</link><guid>100</guid></item></channel></rss>`
	p := &Provider{HTTP: &httpx.Client{HTTP: srv.Client()}, Base: srv.URL}
	var got []provider.Listing
	if err := p.Search(context.Background(), []string{"golang"}, 1, func(l provider.Listing) {
		got = append(got, l)
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0].Vacancy.Description, "удал") {
		t.Fatalf("listing = %#v", got)
	}
	full, err := got[0].Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(full.Description, "relocation") || full.URL != srv.URL+"/vacancies/100" {
		t.Fatalf("detail = %#v", full)
	}
}
