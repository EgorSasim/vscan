package pythonjobs

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestSearch(t *testing.T) {
	const body = `<?xml version="1.0"?><rss version="2.0"><channel><item><title>Senior Engineer, tem</title><link>https://www.python.org/jobs/8139/</link><description>Remote (UK)&lt;p&gt;Python daily.&lt;/p&gt;</description></item></channel></rss>`
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
	if len(got) != 1 || got[0].Source != "python" || got[0].Remote != "yes" || got[0].URL != "https://www.python.org/jobs/8139/" {
		t.Fatalf("%#v", got)
	}
	if !strings.Contains(got[0].Description, "Python daily") {
		t.Fatalf("%#v", got[0])
	}
}
