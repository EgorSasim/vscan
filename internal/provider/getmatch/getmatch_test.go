package getmatch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestOffers(t *testing.T) {
	const body = `{"meta":{"total":1,"offset":0,"limit":100},"offers":[{"id":36107,"is_active":true,"position":"Senior Angular","url":"/vacancies/36107-senior-angular","offer_description":"<b>relocate</b> welcome","company":{"name":"Альфа"},"location_requirements":[{"format":"remote"}]}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/offers" {
			t.Errorf("path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	p := &Provider{HTTP: &httpx.Client{HTTP: srv.Client()}, Base: srv.URL}
	var got []provider.Vacancy
	if err := p.Search(context.Background(), []string{"Angular"}, 1, func(l provider.Listing) {
		got = append(got, l.Vacancy)
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].URL != srv.URL+"/vacancies/36107-senior-angular" || got[0].Company != "Альфа" || got[0].Tags[0] != "remote" {
		t.Fatalf("%#v", got)
	}
}
