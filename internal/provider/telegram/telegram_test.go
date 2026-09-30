package telegram

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"vscan/internal/httpx"
	"vscan/internal/provider"
)

func TestParseAndSearch(t *testing.T) {
	const page = `<div class="tgme_widget_message_wrap"><div class="tgme_widget_message" data-post="remoteit/13942"><div class="tgme_widget_message_text js-message_text" dir="auto">Senior Angular <a href="https://example.com/job">role</a> #remote</div><time datetime="2026-09-24T09:15:19+00:00" class="time"></time></div></div>`
	posts := parsePosts(page)
	if len(posts) != 1 || posts[0].ID != 13942 || posts[0].URL != "https://t.me/remoteit/13942" || posts[0].Remote != "yes" || posts[0].Posted.IsZero() {
		t.Fatalf("%#v", posts)
	}
	if posts[0].Description != "Senior Angular role #remote" {
		t.Fatalf("%q", posts[0].Description)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/s/remoteit" {
			t.Errorf("path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(page))
	}))
	defer srv.Close()
	p := &Provider{HTTP: &httpx.Client{HTTP: srv.Client()}, Channels: []string{"@remoteit"}}
	// Point the preview host at the test server by fetching through a wrapper.
	p.HTTP = prefixGetter{base: srv.URL, inner: p.HTTP}
	var got []provider.Vacancy
	if err := p.Search(context.Background(), nil, 1, func(l provider.Listing) {
		got = append(got, l.Vacancy)
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Source != "telegram" {
		t.Fatalf("%#v", got)
	}
}

func TestDefaultChannels(t *testing.T) {
	want := []string{
		"it_remote", "remote_jobs_ru", "itjobsfeed", "devvacancy", "remoteit",
		"remoteok", "java_vacancies", "python_vacancies", "qa_vacancies", "devops_vacancies",
	}
	if len(Channels) != len(want) {
		t.Fatalf("%v", Channels)
	}
	for i, name := range want {
		if Channels[i] != name {
			t.Fatalf("%v", Channels)
		}
	}
}

func TestEmptyList(t *testing.T) {
	p := &Provider{Channels: []string{}}
	if err := p.Search(context.Background(), nil, 0, func(provider.Listing) {
		t.Fatal("empty list must not emit")
	}); err != nil {
		t.Fatal(err)
	}
}

type prefixGetter struct {
	base  string
	inner Getter
}

func (g prefixGetter) Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error) {
	const host = "https://t.me"
	if len(rawURL) >= len(host) && rawURL[:len(host)] == host {
		rawURL = g.base + rawURL[len(host):]
	}
	return g.inner.Get(ctx, rawURL, header)
}
