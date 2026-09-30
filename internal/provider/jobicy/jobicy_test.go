package jobicy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestParse(t *testing.T) {
	const body = `{"jobs":[{"id":1,"url":"https://jobicy.com/jobs/1-go","jobTitle":"Go Developer","companyName":"Acme","jobExcerpt":"Senior Go","jobGeo":"USA","pubDate":"2026-09-29T14:32:46+00:00","salaryMin":100,"salaryMax":200,"salaryCurrency":"USD","salaryPeriod":"yearly"}]}`
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
	if len(got) != 1 || got[0].Salary != "100-200 USD yearly" || got[0].Remote != "yes" {
		t.Fatalf("%#v", got)
	}
	if got[0].Posted.IsZero() || got[0].Location != "USA" {
		t.Fatalf("%#v", got[0])
	}
}
