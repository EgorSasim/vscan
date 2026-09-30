package larajobs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestSearch(t *testing.T) {
	const body = `<?xml version="1.0"?><rss version="2.0" xmlns:job="https://larajobs.com/ns"><channel><item><title>Senior Laravel Engineer</title><link>https://larajobs.com/job/3939</link><pubDate>Fri, 25 Sep 2026 20:52:41 +0000</pubDate><job:location>Remote/USA</job:location><job:job_type>FULL_TIME</job:job_type><job:salary>$160,000.00 - $180,000.00</job:salary><job:company>Evolution Collect, LLC</job:company><job:tags>AWS,Laravel,PHP</job:tags></item></channel></rss>`
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
	if v.Source != "larajobs" || v.Company != "Evolution Collect, LLC" || v.Remote != "yes" || v.Location != "Remote/USA" {
		t.Fatalf("%#v", v)
	}
	if v.Posted.IsZero() || len(v.Skills) != 3 || v.Salary == "" {
		t.Fatalf("%#v", v)
	}
}
