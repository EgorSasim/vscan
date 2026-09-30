package himalayas

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestParse(t *testing.T) {
	const body = `{"nextCursor":"","jobs":[{"title":"Go Lead","companyName":"Acme","excerpt":"Senior Go","guid":"https://himalayas.app/companies/acme/jobs/go-lead","pubDate":1790741513,"minSalary":null,"maxSalary":null,"currency":null,"seniority":["Senior"],"locationRestrictions":["USA"]}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	if len(got) != 1 || got[0].Remote != "yes" || got[0].Location != "USA" || got[0].Posted.IsZero() {
		t.Fatalf("%#v", got)
	}
	if got[0].URL != "https://himalayas.app/companies/acme/jobs/go-lead" {
		t.Fatal(got[0].URL)
	}
}
