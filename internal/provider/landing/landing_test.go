package landing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestParse(t *testing.T) {
	const body = `[{"title":"Senior Angular Developer","url":"https://landing.jobs/at/in-scale/senior-angular","remote":true,"type":"Full-time","currency_code":"EUR","gross_salary_low":50000,"gross_salary_high":67000,"published_at":"2026-09-01T09:38:38.127Z","role_description":"<p>TypeScript</p>","main_requirements":null,"nice_to_have":null,"tags":["Angular","TypeScript"],"locations":[{"city":"Lisbon","country_code":"PT"}]}]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("offset") != "0" {
			t.Errorf("offset %s", r.URL.Query().Get("offset"))
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
	if v.Company != "in scale" || v.Remote != "yes" || v.Salary != "50000-67000 EUR" || v.Location != "Lisbon, PT" {
		t.Fatalf("%#v", v)
	}
	if v.Posted.IsZero() || v.Description != "TypeScript" {
		t.Fatalf("%#v", v)
	}
}
