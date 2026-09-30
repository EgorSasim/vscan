package arbeitnow

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestParse(t *testing.T) {
	const body = `{"data":[{"slug":"go-berlin-1","company_name":"Acme","title":"Go Developer","description":"<p>Senior Go</p>","remote":false,"location":"Berlin","created_at":1786516800,"tags":["go"]}],"links":{"next":null}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") != "1" {
			t.Errorf("page = %s", r.URL.Query().Get("page"))
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
	if len(got) != 1 || got[0].Title != "Go Developer" || got[0].Remote != "no" || got[0].Location != "Berlin" {
		t.Fatalf("%#v", got)
	}
	if got[0].URL != "https://www.arbeitnow.com/view/go-berlin-1" || got[0].Posted.IsZero() {
		t.Fatalf("%#v", got[0])
	}
}
