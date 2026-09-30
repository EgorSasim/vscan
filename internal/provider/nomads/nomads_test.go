package nomads

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestParse(t *testing.T) {
	const body = `[{"url":"https://www.workingnomads.com/job/go/1/","title":"Go Developer","description":"<p>Senior Go</p>","company_name":"Acme","category_name":"Development","tags":"go, remote","location":"Germany","pub_date":"2026-09-29T13:14:42-04:00"}]`
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
	if len(got) != 1 || got[0].Remote != "yes" || got[0].Location != "Germany" || got[0].Posted.IsZero() {
		t.Fatalf("%#v", got)
	}
	if got[0].Company != "Acme" || len(got[0].Tags) < 2 {
		t.Fatalf("%#v", got[0])
	}
}
