package elixirjobs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestSearch(t *testing.T) {
	const body = `<?xml version="1.0"?><rss version="2.0"><channel><item><title><![CDATA[Elixir Software Engineer ]]></title><description><![CDATA[Job place: Remote<br/>Driftrock is hiring.]]></description><pubDate>14 Sep 2026 21:47:44 +0000</pubDate><link><![CDATA[https://elixirjobs.net/offers/driftrock]]></link></item></channel></rss>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	p := &Provider{HTTP: &httpx.Client{HTTP: srv.Client()}, URL: srv.URL}
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
	if v.Source != "elixir" || v.Remote != "yes" || v.URL != "https://elixirjobs.net/offers/driftrock" || v.Posted.Day() != 14 {
		t.Fatalf("%#v", v)
	}
	if v.Title != "Elixir Software Engineer" {
		t.Fatalf("%q", v.Title)
	}
}
