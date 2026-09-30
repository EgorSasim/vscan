package nofluff

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestParse(t *testing.T) {
	const body = `{"totalPages":1,"postings":[{"name":"Acme","title":"Senior Angular Developer","url":"senior-angular-acme","posted":1790233428592,"seniority":["Senior"],"technology":["TypeScript"],"fullyRemote":true,"regions":["pl"],"location":{"places":[{"city":"Warsaw","country":{"name":"Poland"}}]},"salary":{"from":20000,"to":28000,"currency":"PLN","period":"Month","type":"b2b"},"tiles":{"values":[{"value":"Angular","type":"requirement"}]}}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/infiniteSearch+json" {
			t.Errorf("content-type %s", r.Header.Get("Content-Type"))
		}
		if r.URL.Query().Get("pageTo") != "1" {
			t.Errorf("pageTo %s", r.URL.Query().Get("pageTo"))
		}
		got, _ := io.ReadAll(r.Body)
		if string(got) != `{"criteriaSearch":{}}` {
			t.Errorf("body %s", got)
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
	if v.URL != "https://nofluffjobs.com/job/senior-angular-acme" || v.Remote != "yes" || v.Salary != "20000-28000 PLN Month" {
		t.Fatalf("%#v", v)
	}
	if v.Company != "Acme" || v.Location != "Warsaw" || v.Posted.IsZero() {
		t.Fatalf("%#v", v)
	}
}
