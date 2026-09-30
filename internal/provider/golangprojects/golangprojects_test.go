package golangprojects

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestSearch(t *testing.T) {
	const body = `<?xml version="1.0"?><rss version="2.0"><channel><item><title>Sr. Software Engineer&#160;@&#160;Prenosis</title><link>https://www.golangprojects.com/golang-go-job-example.html</link><description>Remote - Go and Kubernetes.</description></item></channel></rss>`
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
	if v.Source != "golangprojects" || v.Title != "Sr. Software Engineer" || v.Company != "Prenosis" || v.Remote != "yes" {
		t.Fatalf("%#v", v)
	}
}
