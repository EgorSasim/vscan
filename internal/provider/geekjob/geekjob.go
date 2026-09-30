package geekjob

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const defaultBase = "https://geekjob.ru"

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider uses Geekjob's public vacancy JSON and the vacancy HTML page.
type Provider struct {
	HTTP Getter
	Base string
}

func (p *Provider) Name() string { return "geekjob" }

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
	for page := 1; maxPages == 0 || page <= maxPages; page++ {
		u, err := url.Parse(p.base() + "/json/find/vacancy")
		if err != nil {
			return err
		}
		q := u.Query()
		q.Set("page", fmt.Sprint(page))
		if hint != "" {
			q.Set("qs", hint)
		}
		u.RawQuery = q.Encode()
		body, err := p.HTTP.Get(ctx, u.String(), nil)
		if err != nil {
			return err
		}
		batch, next, err := parse(body, p.base())
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			return nil
		}
		for _, v := range batch {
			v := v
			emit(provider.Listing{
				Vacancy: v,
				Fetch: func(ctx context.Context) (provider.Vacancy, error) {
					return p.detail(ctx, v)
				},
			})
		}
		if next == 0 || next == page {
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
	if block := htmlutil.ClassBlock(string(body), "vacancy-description"); block != "" {
		v.Description = htmlutil.Text(block)
	}
	return v, nil
}

func parse(body []byte, origin string) ([]provider.Vacancy, int, error) {
	var raw struct {
		Nextpage int `json:"nextpage"`
		Data     []struct {
			ID       string `json:"id"`
			Position string `json:"position"`
			Company  struct {
				Name string `json:"name"`
			} `json:"company"`
			JobFormat struct {
				Remote   bool `json:"remote"`
				Relocate bool `json:"relocate"`
			} `json:"jobFormat"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, 0, err
	}
	var out []provider.Vacancy
	for _, it := range raw.Data {
		if it.ID == "" {
			continue
		}
		var tags []string
		if it.JobFormat.Remote {
			tags = append(tags, "remote")
		}
		if it.JobFormat.Relocate {
			tags = append(tags, "relocate")
		}
		out = append(out, provider.Vacancy{
			URL:     origin + "/vacancy/" + it.ID,
			Title:   htmlutil.Text(it.Position),
			Company: htmlutil.Text(it.Company.Name),
			Source:  "geekjob",
			Tags:    tags,
		})
	}
	return out, raw.Nextpage, nil
}
