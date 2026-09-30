package remoteok

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestSkipsLegalNotice(t *testing.T) {
	const body = `[{"legal":"please link back"},{"id":1,"position":"Angular Dev","company":"Acme","url":"https://remoteok.com/remote-jobs/1","description":"<p>Senior Angular</p>","tags":["angular","dev"]}]`
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
	if len(got) != 1 || got[0].Title != "Angular Dev" || got[0].Tags[len(got[0].Tags)-1] != "remote" {
		t.Fatalf("%#v", got)
	}
}
