package jobspresso

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestParse(t *testing.T) {
	const body = `<?xml version="1.0" encoding="UTF-8"?><rss version="2.0" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:content="http://purl.org/rss/1.0/modules/content/"><channel><item><title>Senior Angular Developer</title><link>https://jobspresso.co/job/senior-angular/</link><dc:creator>Acme&lt;br&gt;⚲&amp;nbsp;Europe</dc:creator><content:encoded>&lt;p&gt;TypeScript&lt;/p&gt;</content:encoded><pubDate>Sat, 29 Aug 2026 02:12:12 +0000</pubDate></item></channel></rss>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("paged") != "1" {
			t.Errorf("paged %s", r.URL.Query().Get("paged"))
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
	if v.Company != "Acme" || v.Location != "Europe" || v.Remote != "yes" || v.Description != "TypeScript" {
		t.Fatalf("%#v", v)
	}
	if v.Posted.IsZero() {
		t.Fatalf("%#v", v)
	}
}
