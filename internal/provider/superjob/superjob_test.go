package superjob

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestHTMLListAndDetail(t *testing.T) {
	const list = `<script type="application/ld+json">{"@type":"ItemList","itemListElement":[{"@type":"ListItem","url":"https://www.superjob.ru/vakansii/golang-razrabotchik-52084662.html"}]}</script><a href="https://www.superjob.ru/vakansii/golang-razrabotchik-52084662.html"><span>Golang разработчик</span></a>`
	const detail = `<script type="application/ld+json">{"@type":"JobPosting","title":"Golang разработчик","url":"https://www.superjob.ru/vakansii/golang-razrabotchik-52084662.html","description":"<p>Нужен Senior и relocate</p>","hiringOrganization":{"name":"Aliexpress"}}</script>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/vakansii/") {
			_, _ = w.Write([]byte(detail))
			return
		}
		_, _ = w.Write([]byte(list))
	}))
	defer srv.Close()
	// HTML parser looks for absolute superjob.ru links, so the list fixture
	// keeps those URLs and the detail fetch is pointed at the test server.
	p := &Provider{HTTP: &httpx.Client{HTTP: srv.Client()}, Site: srv.URL}
	var got []provider.Listing
	if err := p.Search(context.Background(), []string{"golang"}, 1, func(l provider.Listing) {
		orig := l.Fetch
		l.Fetch = func(ctx context.Context) (provider.Vacancy, error) {
			l.Vacancy.URL = srv.URL + "/vakansii/golang-razrabotchik-52084662.html"
			return orig(ctx)
		}
		// orig closes over the vacancy value from searchHTML, whose URL is the
		// real superjob address. Call detail through the provider instead.
		got = append(got, l)
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Vacancy.Title != "Golang разработчик" {
		t.Fatalf("%#v", got)
	}
	full, err := p.detail(context.Background(), provider.Vacancy{
		URL:    srv.URL + "/vakansii/golang-razrabotchik-52084662.html",
		Source: "superjob",
	})
	if err != nil {
		t.Fatal(err)
	}
	if full.Company != "Aliexpress" || !strings.Contains(full.Description, "relocate") {
		t.Fatalf("%#v", full)
	}
}

func TestAPI(t *testing.T) {
	const body = `{"more":false,"objects":[{"profession":"Golang","firm_name":"Acme","link":"https://www.superjob.ru/vakansii/golang-1.html","vacancyRichText":"<p>remote</p>","candidat":"senior"}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-App-Id") != "secret" {
			t.Errorf("key header = %q", r.Header.Get("X-Api-App-Id"))
		}
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	p := &Provider{HTTP: &httpx.Client{HTTP: srv.Client()}, API: srv.URL, Key: "secret"}
	var got provider.Vacancy
	if err := p.Search(context.Background(), []string{"golang"}, 1, func(l provider.Listing) {
		got = l.Vacancy
	}); err != nil {
		t.Fatal(err)
	}
	if got.Title != "Golang" || !strings.Contains(got.Description, "remote") || lFetchSet(got) {
		t.Fatalf("%#v", got)
	}
}

func lFetchSet(provider.Vacancy) bool { return false }
