package djinni

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const defaultBase = "https://djinni.co"

var jobHref = regexp.MustCompile(`(?i)href=["']([^"']*/jobs/(\d+)[^"']*)["']`)

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider reads the public Djinni jobs HTML.
// An empty body is reported as an error so other boards keep running.
type Provider struct {
	HTTP Getter
	Base string
}

func (p *Provider) Name() string { return "djinni" }

func (p *Provider) base() string {
	if p.Base != "" {
		return strings.TrimRight(p.Base, "/")
	}
	return defaultBase
}

func (p *Provider) Search(ctx context.Context, hints []string, maxPages int, emit func(provider.Listing)) error {
	if len(hints) == 0 {
		hints = []string{""}
	}
	var first error
	for _, hint := range hints {
		if err := p.searchHint(ctx, hint, maxPages, emit); err != nil && first == nil {
			first = err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return first
}

func (p *Provider) searchHint(ctx context.Context, hint string, maxPages int, emit func(provider.Listing)) error {
	limit := maxPages
	if limit == 0 {
		limit = 10
	}
	seen := map[string]struct{}{}
	for page := 1; page <= limit; page++ {
		u, err := url.Parse(p.base() + "/jobs/")
		if err != nil {
			return err
		}
		q := u.Query()
		q.Set("search", hint)
		if page > 1 {
			q.Set("page", fmt.Sprint(page))
		}
		u.RawQuery = q.Encode()
		body, err := p.HTTP.Get(ctx, u.String(), nil)
		if err != nil {
			return err
		}
		if len(bytes.TrimSpace(body)) == 0 {
			if page == 1 {
				return fmt.Errorf("пустой ответ, площадка закрыла выдачу")
			}
			return nil
		}
		found := parseList(string(body), p.base())
		if len(found) == 0 {
			return nil
		}
		fresh := 0
		for _, v := range found {
			if _, ok := seen[v.URL]; ok {
				continue
			}
			seen[v.URL] = struct{}{}
			fresh++
			v := v
			emit(provider.Listing{
				Vacancy: v,
				Fetch: func(ctx context.Context) (provider.Vacancy, error) {
					return p.detail(ctx, v)
				},
			})
		}
		if fresh == 0 {
			return nil
		}
	}
	return nil
}

func (p *Provider) detail(ctx context.Context, v provider.Vacancy) (provider.Vacancy, error) {
	body, err := p.HTTP.Get(ctx, v.URL, nil)
	if err != nil {
		return provider.Vacancy{}, err
	}
	page := string(body)
	for _, class := range []string{"job-post__description", "vacancy-description", "job_description"} {
		if block := htmlutil.ClassBlock(page, class); block != "" {
			v.Description = htmlutil.Text(block)
			break
		}
	}
	if title := htmlutil.AnchorText(page, v.URL); title != "" && v.Title == "" {
		v.Title = title
	}
	return v, nil
}

func parseList(page, origin string) []provider.Vacancy {
	matches := jobHref.FindAllStringSubmatch(page, -1)
	seen := map[string]struct{}{}
	var out []provider.Vacancy
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		link := provider.Abs(origin, htmlUnescape(m[1]))
		if _, ok := seen[link]; ok {
			continue
		}
		seen[link] = struct{}{}
		title := htmlutil.AnchorText(page, m[1])
		if title == "" {
			title = htmlutil.AnchorText(page, link)
		}
		out = append(out, provider.Vacancy{
			URL:    link,
			Title:  title,
			Source: "djinni",
		})
	}
	return out
}

func htmlUnescape(s string) string {
	return strings.NewReplacer("&amp;", "&").Replace(s)
}
