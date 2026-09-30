package fourday

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestParse(t *testing.T) {
	const body = `{"has_more":false,"jobs":[{"title":"Senior Angular Developer","slug":"senior-angular-at-acme","company_name":"Acme","work_arrangement":"remote","posted":1790729109,"category":"engineering","level":"senior","schedule_type":"4_day_week","is_expired":false,"stack":[{"name":"TypeScript"},{"name":"Angular"}],"locations":[{"city":"Lisbon","country":"Portugal","work_arrangement":"remote"}]}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") != "1" {
			t.Errorf("page %s", r.URL.Query().Get("page"))
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
	if v.URL != "https://4dayweek.io/job/senior-angular-at-acme" || v.Remote != "yes" || v.Location != "Lisbon, Portugal" {
		t.Fatalf("%#v", v)
	}
	if v.Posted.IsZero() || v.Company != "Acme" {
		t.Fatalf("%#v", v)
	}
}
