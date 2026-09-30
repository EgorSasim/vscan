package engine

import (
	"context"
	"sync"
	"testing"
	"time"

	"vscan/internal/emit"
	"vscan/internal/provider"
	"vscan/internal/query"
	"vscan/internal/store"
)

type fake struct {
	name    string
	items   []provider.Listing
	release <-chan struct{}
}

func (f fake) Name() string { return f.name }

func (f fake) Search(ctx context.Context, hints []string, maxPages int, emit func(provider.Listing)) error {
	_ = hints
	_ = maxPages
	for i, item := range f.items {
		if i == 1 && f.release != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-f.release:
			}
		}
		emit(item)
	}
	return nil
}

type memOut struct {
	mu   sync.Mutex
	hits []emit.Hit
	got  chan struct{}
}

func (m *memOut) Emit(h emit.Hit) error {
	m.mu.Lock()
	m.hits = append(m.hits, h)
	m.mu.Unlock()
	if m.got != nil {
		select {
		case m.got <- struct{}{}:
		default:
		}
	}
	return nil
}

func TestStreamsBeforeProviderFinishes(t *testing.T) {
	expr, err := query.Parse("Senior&|Angular&|Remote")
	if err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	out := &memOut{got: make(chan struct{}, 1)}
	prov := fake{
		name: "fake",
		items: []provider.Listing{
			{Vacancy: provider.Vacancy{
				URL:         "https://example.com/1",
				Title:       "Senior Angular Developer",
				Description: "we can relocate",
				Source:      "fake",
			}},
			{Vacancy: provider.Vacancy{
				URL:    "https://example.com/2",
				Title:  "Junior Go",
				Source: "fake",
			}},
		},
		release: release,
	}
	done := make(chan int, 1)
	go func() {
		n, err := search(context.Background(), Config{
			Expr:      expr,
			Providers: []provider.Provider{prov},
			Out:       out,
		}, false)
		if err != nil {
			t.Errorf("search: %v", err)
		}
		done <- n
	}()
	select {
	case <-out.got:
	case <-time.After(2 * time.Second):
		t.Fatal("first link was not emitted before the provider finished")
	}
	close(release)
	select {
	case n := <-done:
		if n != 1 {
			t.Fatalf("matches = %d", n)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("search did not finish")
	}
}

func TestScannerBaselineThenNew(t *testing.T) {
	expr, err := query.Parse("Go&Developer")
	if err != nil {
		t.Fatal(err)
	}
	seen, err := store.OpenSeen(store.Dirs{Cache: t.TempDir()}, "Go&Developer", []string{"fake"})
	if err != nil {
		t.Fatal(err)
	}
	out := &memOut{}
	job := func(url, title string) provider.Listing {
		return provider.Listing{Vacancy: provider.Vacancy{URL: url, Title: title, Source: "fake"}}
	}
	cfg := Config{
		Expr: expr,
		Providers: []provider.Provider{fake{name: "fake", items: []provider.Listing{
			job("https://example.com/old", "Go Developer"),
		}}},
		Out:  out,
		Seen: seen,
	}
	n, err := search(context.Background(), cfg, true)
	if err != nil || n != 0 || len(out.hits) != 0 {
		t.Fatalf("baseline n=%d hits=%d err=%v", n, len(out.hits), err)
	}
	if !seen.Has("https://example.com/old") {
		t.Fatal("baseline did not cache")
	}
	cfg.Providers = []provider.Provider{fake{name: "fake", items: []provider.Listing{
		job("https://example.com/old", "Go Developer"),
		job("https://example.com/new", "Go Developer"),
	}}}
	n, err = search(context.Background(), cfg, false)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 || len(out.hits) != 1 || out.hits[0].URL != "https://example.com/new" {
		t.Fatalf("n=%d hits=%#v", n, out.hits)
	}
}

func TestDetailFetchWhenSnippetIsNotEnough(t *testing.T) {
	expr, err := query.Parse("Senior&|Angular&|Remote")
	if err != nil {
		t.Fatal(err)
	}
	out := &memOut{}
	prov := fake{name: "fake", items: []provider.Listing{{
		Vacancy: provider.Vacancy{
			URL:    "https://example.com/1",
			Title:  "Senior Angular Developer",
			Source: "fake",
		},
		Fetch: func(context.Context) (provider.Vacancy, error) {
			return provider.Vacancy{
				URL:         "https://example.com/1",
				Title:       "Senior Angular Developer",
				Description: "relocation bonus",
				Source:      "fake",
			}, nil
		},
	}}}
	n, err := search(context.Background(), Config{
		Expr:      expr,
		Providers: []provider.Provider{prov},
		Out:       out,
	}, false)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}
