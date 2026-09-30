// Package httpx is a shared HTTP client with per-host pacing and retries.
package httpx

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const UserAgent = "vscan/0.1 (personal vacancy scanner)"

// Client wraps net/http with a polite per-host rate and a few retries.
type Client struct {
	HTTP    *http.Client
	limiter *limiter
}

func New() *Client {
	return &Client{
		HTTP: &http.Client{
			Timeout: 25 * time.Second,
		},
		limiter: newLimiter(2, 4),
	}
}

// Get returns the response body for a 2xx reply.
func (c *Client) Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error) {
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		if c.limiter != nil {
			if err := c.limiter.Wait(ctx, hostOf(rawURL)); err != nil {
				return nil, err
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", UserAgent)
		req.Header.Set("Accept", "application/json, application/xml, text/html, */*")
		for k, v := range header {
			req.Header.Set(k, v)
		}
		resp, err := c.HTTP.Do(req)
		if err != nil {
			last = err
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if !sleep(ctx, backoff(attempt)) {
				return nil, ctx.Err()
			}
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if readErr != nil {
			last = readErr
			continue
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return body, nil
		}
		last = fmt.Errorf("http %d", resp.StatusCode)
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			wait := backoff(attempt)
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if sec, err := strconv.Atoi(ra); err == nil && sec > 0 && sec <= 30 {
					wait = time.Duration(sec) * time.Second
				}
			}
			if !sleep(ctx, wait) {
				return nil, ctx.Err()
			}
			continue
		}
		return nil, last
	}
	return nil, last
}

func hostOf(rawURL string) string {
	rest := rawURL
	if i := strings.Index(rest, "://"); i >= 0 {
		rest = rest[i+3:]
	}
	if i := strings.IndexAny(rest, "/?#"); i >= 0 {
		rest = rest[:i]
	}
	return strings.ToLower(rest)
}

func backoff(attempt int) time.Duration {
	return time.Duration(attempt+1) * 400 * time.Millisecond
}

func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

// limiter is a token bucket keyed by host. rate is tokens per second.
type limiter struct {
	mu    sync.Mutex
	rate  float64
	burst float64
	hosts map[string]*bucket
}

type bucket struct {
	tokens float64
	last   time.Time
}

func newLimiter(rate, burst float64) *limiter {
	return &limiter{rate: rate, burst: burst, hosts: map[string]*bucket{}}
}

func (l *limiter) Wait(ctx context.Context, host string) error {
	for {
		l.mu.Lock()
		b := l.hosts[host]
		now := time.Now()
		if b == nil {
			b = &bucket{tokens: l.burst - 1, last: now}
			l.hosts[host] = b
			l.mu.Unlock()
			return nil
		}
		elapsed := now.Sub(b.last).Seconds()
		b.tokens += elapsed * l.rate
		if b.tokens > l.burst {
			b.tokens = l.burst
		}
		b.last = now
		if b.tokens >= 1 {
			b.tokens--
			l.mu.Unlock()
			return nil
		}
		need := (1 - b.tokens) / l.rate
		l.mu.Unlock()
		if !sleep(ctx, time.Duration(need*float64(time.Second))+time.Millisecond) {
			return ctx.Err()
		}
	}
}
