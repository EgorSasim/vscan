package remotive

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestParse(t *testing.T) {
	const body = `{"00-warning":"moved","jobs":[{"id":1,"url":"https://remotive.com/remote-jobs/go-1","title":"Go Engineer","company_name":"Acme","description":"<p>Senior</p>","candidate_required_location":"USA","publication_date":"2026-09-21T12:55:11","salary":"","tags":["go"],"job_type":"full_time"}]}`
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
	if len(got) != 1 || got[0].Remote != "yes" || got[0].Location != "USA" || got[0].Posted.IsZero() {
		t.Fatalf("%#v", got)
	}
}
