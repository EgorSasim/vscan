package wwr

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestParseItem(t *testing.T) {
	const body = `<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"><channel><link>https://weworkremotely.com/categories/remote-programming-jobs.rss</link><item><title>Toptal: Senior Angular Developer</title><region>Anywhere in the World</region><category>Full-Stack Programming</category><description>&lt;p&gt;Remote Angular work&lt;/p&gt;</description><link>https://weworkremotely.com/remote-jobs/toptal-senior-angular</link><guid>https://weworkremotely.com/remote-jobs/toptal-senior-angular</guid></item></channel></rss>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	p := &Provider{HTTP: &httpx.Client{HTTP: srv.Client()}, URL: srv.URL}
	var n int
	err := p.Search(context.Background(), nil, 0, func(l provider.Listing) {
		n++
		if l.Company != "Toptal" || !strings.Contains(l.Title, "Angular") || !strings.Contains(l.Description, "Remote Angular") {
			t.Fatalf("%#v", l)
		}
	})
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}
