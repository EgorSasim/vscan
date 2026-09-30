// Package engine runs providers in parallel and prints matches as they arrive.
package engine

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"vscan/internal/emit"
	"vscan/internal/provider"
	"vscan/internal/query"
	"vscan/internal/store"
)

// Config is one search pass.
type Config struct {
	Expr      query.Expr
	Syn       *query.Synonyms
	Providers []provider.Provider
	Hints     []string
	MaxPages  int
	Out       emit.Emitter
	Seen      *store.Seen
	Logf      func(string, ...any)
	Progress  func(string, ...any)
}

// Run repeats the search when Scanner is set.
// The first pass is silent only when the seen-cache was empty at start.
func Run(ctx context.Context, cfg Config, scanner bool, every time.Duration) int {
	baseline := scanner && cfg.Seen != nil && cfg.Seen.InitialEmpty()
	if baseline && cfg.Progress != nil {
		cfg.Progress("первый проход запоминает текущие ссылки, уведомления начнутся со следующего")
	}
	for {
		n, err := search(ctx, cfg, baseline)
		if errors.Is(err, emit.ErrPipe) {
			return 0
		}
		if ctx.Err() != nil {
			if scanner || n > 0 {
				return 0
			}
			return 1
		}
		if !scanner {
			if n > 0 {
				return 0
			}
			return 1
		}
		if cfg.Progress != nil {
			if baseline {
				cfg.Progress("база сохранена, дальше печатаются только новые ссылки")
			} else {
				cfg.Progress("новых ссылок: %d, следующий проход через %s", n, every)
			}
		}
		baseline = false
		timer := time.NewTimer(every)
		select {
		case <-ctx.Done():
			timer.Stop()
			return 0
		case <-timer.C:
		}
	}
}

func search(ctx context.Context, cfg Config, baseline bool) (int, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logf := cfg.Logf
	if logf == nil {
		logf = func(string, ...any) {}
	}
	sink := &sink{
		out:      cfg.Out,
		seen:     cfg.Seen,
		baseline: baseline,
		reserved: map[string]struct{}{},
		emitted:  map[string]struct{}{},
	}

	listings := make(chan provider.Listing, 32)
	var pwg sync.WaitGroup
	for _, p := range cfg.Providers {
		pwg.Add(1)
		go func(p provider.Provider) {
			defer pwg.Done()
			err := p.Search(ctx, cfg.Hints, cfg.MaxPages, func(l provider.Listing) {
				select {
				case <-ctx.Done():
				case listings <- l:
				}
			})
			if err != nil && ctx.Err() == nil {
				logf("%s: %v", p.Name(), err)
			}
		}(p)
	}
	go func() {
		pwg.Wait()
		close(listings)
	}()

	jobs := make(chan provider.Listing, 32)
	var dwg sync.WaitGroup
	for i := 0; i < 8; i++ {
		dwg.Add(1)
		go func() {
			defer dwg.Done()
			for l := range jobs {
				if l.Fetch == nil || ctx.Err() != nil {
					continue
				}
				v, err := l.Fetch(ctx)
				if err != nil {
					if ctx.Err() == nil {
						logf("%s: %v", l.Vacancy.Source, err)
					}
					continue
				}
				if !query.Match(cfg.Expr, v.Document(), cfg.Syn) {
					continue
				}
				if err := sink.deliver(v); err != nil {
					if errors.Is(err, emit.ErrPipe) {
						cancel()
						continue
					}
					logf("%v", err)
				}
			}
		}()
	}

	var pipe error
	for l := range listings {
		if ctx.Err() != nil {
			break
		}
		url := provider.Canonical(l.Vacancy.URL)
		if url == "" {
			continue
		}
		l.Vacancy.URL = url
		sink.mu.Lock()
		if _, ok := sink.reserved[url]; ok {
			sink.mu.Unlock()
			continue
		}
		matched := query.Match(cfg.Expr, l.Vacancy.Document(), cfg.Syn)
		if matched || l.Fetch != nil {
			sink.reserved[url] = struct{}{}
		}
		sink.mu.Unlock()
		if matched {
			if err := sink.deliver(l.Vacancy); err != nil {
				if errors.Is(err, emit.ErrPipe) {
					pipe = err
					cancel()
					break
				}
				logf("%v", err)
			}
			continue
		}
		if l.Fetch != nil {
			select {
			case <-ctx.Done():
			case jobs <- l:
			}
		}
	}
	close(jobs)
	dwg.Wait()
	sink.mu.Lock()
	n := sink.count
	sink.mu.Unlock()
	if pipe != nil {
		return n, pipe
	}
	return n, nil
}

type sink struct {
	mu       sync.Mutex
	out      emit.Emitter
	seen     *store.Seen
	baseline bool
	reserved map[string]struct{}
	emitted  map[string]struct{}
	count    int
}

func (s *sink) deliver(v provider.Vacancy) error {
	url := provider.Canonical(v.URL)
	if url == "" {
		return nil
	}
	if s.seen != nil && s.seen.Has(url) {
		return nil
	}
	if s.baseline {
		if s.seen != nil {
			return s.seen.Add(url)
		}
		return nil
	}
	s.mu.Lock()
	if _, ok := s.emitted[url]; ok {
		s.mu.Unlock()
		return nil
	}
	s.emitted[url] = struct{}{}
	s.mu.Unlock()
	if s.out != nil {
		if err := s.out.Emit(emit.Hit{
			URL:     url,
			Title:   v.Title,
			Company: v.Company,
			Source:  v.Source,
		}); err != nil {
			s.mu.Lock()
			delete(s.emitted, url)
			s.mu.Unlock()
			return err
		}
	}
	s.mu.Lock()
	s.count++
	s.mu.Unlock()
	if s.seen != nil {
		if err := s.seen.Add(url); err != nil {
			return fmt.Errorf("кэш: %w", err)
		}
	}
	return nil
}
