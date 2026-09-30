package muse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const defaultURL = "https://www.themuse.com/api/public/jobs?category=Software+Engineering"

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider reads The Muse public jobs API, software engineering category.
// An unlimited search stops after five pages.
type Provider struct {
	HTTP Getter
	URL  string
}

func (p *Provider) Name() string { return "muse" }

func (p *Provider) Search(ctx context.Context, hints []string, maxPages int, emit func(provider.Listing)) error {
	_ = hints
	endpoint := p.URL
	if endpoint == "" {
		endpoint = defaultURL
	}
	seen := map[string]struct{}{}
	for page := 1; ; page++ {
		if maxPages > 0 && page > maxPages {
			return nil
		}
		if maxPages == 0 && page > 5 {
			return nil
		}
		u, err := url.Parse(endpoint)
		if err != nil {
			return err
		}
		q := u.Query()
		q.Set("page", fmt.Sprint(page))
		u.RawQuery = q.Encode()
		body, err := p.HTTP.Get(ctx, u.String(), nil)
		if err != nil {
			return err
		}
		jobs, pages, err := parse(body)
		if err != nil {
			return err
		}
		added := 0
		for _, job := range jobs {
			if _, ok := seen[job.URL]; ok {
				continue
			}
			seen[job.URL] = struct{}{}
			emit(provider.Listing{Vacancy: job})
			added++
		}
		if added == 0 || (pages > 0 && page >= pages) {
			return nil
		}
	}
}

func parse(body []byte) ([]provider.Vacancy, int, error) {
	var raw struct {
		PageCount int `json:"page_count"`
		Results   []struct {
			Name      string `json:"name"`
			Contents  string `json:"contents"`
			Published string `json:"publication_date"`
			Company   struct {
				Name string `json:"name"`
			} `json:"company"`
			Refs struct {
				Landing string `json:"landing_page"`
			} `json:"refs"`
			Locations []struct {
				Name string `json:"name"`
			} `json:"locations"`
			Levels []struct {
				Name string `json:"name"`
			} `json:"levels"`
			Categories []struct {
				Name string `json:"name"`
			} `json:"categories"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, 0, err
	}
	var out []provider.Vacancy
	for _, it := range raw.Results {
		if it.Refs.Landing == "" || it.Name == "" {
			continue
		}
		var places []string
		for _, loc := range it.Locations {
			if loc.Name != "" {
				places = append(places, loc.Name)
			}
		}
		var skills []string
		for _, lv := range it.Levels {
			if lv.Name != "" {
				skills = append(skills, lv.Name)
			}
		}
		for _, cat := range it.Categories {
			if cat.Name != "" {
				skills = append(skills, cat.Name)
			}
		}
		out = append(out, provider.Vacancy{
			URL:         it.Refs.Landing,
			Title:       htmlutil.Text(it.Name),
			Company:     htmlutil.Text(it.Company.Name),
			Source:      "muse",
			Description: htmlutil.Text(it.Contents),
			Skills:      skills,
			Location:    strings.Join(places, "; "),
			Posted:      provider.ParseTime(it.Published),
		})
	}
	return out, raw.PageCount, nil
}
