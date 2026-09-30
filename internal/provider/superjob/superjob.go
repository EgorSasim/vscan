package superjob

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const (
	defaultSite = "https://www.superjob.ru"
	defaultAPI  = "https://api.superjob.ru"
)

var vacancyHref = regexp.MustCompile(`https?://(?:www\.)?superjob\.ru/vakansii/[a-z0-9_-]+-\d+\.html`)

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider uses the SuperJob API when Key is set, otherwise the public HTML search.
type Provider struct {
	HTTP Getter
	Key  string
	Site string
	API  string
}

func (p *Provider) Name() string { return "superjob" }

func (p *Provider) Search(ctx context.Context, hints []string, maxPages int, emit func(provider.Listing)) error {
	if len(hints) == 0 {
		hints = []string{""}
	}
	var first error
	for _, hint := range hints {
		var err error
		if p.Key != "" {
			err = p.searchAPI(ctx, hint, maxPages, emit)
		} else {
			err = p.searchHTML(ctx, hint, maxPages, emit)
		}
		if err != nil && first == nil {
			first = err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return first
}

func (p *Provider) api() string {
	if p.API != "" {
		return strings.TrimRight(p.API, "/")
	}
	return defaultAPI
}

func (p *Provider) site() string {
	if p.Site != "" {
		return strings.TrimRight(p.Site, "/")
	}
	return defaultSite
}

func (p *Provider) searchAPI(ctx context.Context, hint string, maxPages int, emit func(provider.Listing)) error {
	for page := 0; maxPages == 0 || page < maxPages; page++ {
		u, err := url.Parse(p.api() + "/2.0/vacancies/")
		if err != nil {
			return err
		}
		q := u.Query()
		q.Set("keyword", hint)
		q.Set("count", "100")
		q.Set("page", fmt.Sprint(page))
		u.RawQuery = q.Encode()
		body, err := p.HTTP.Get(ctx, u.String(), map[string]string{"X-Api-App-Id": p.Key})
		if err != nil {
			return err
		}
		var parsed struct {
			Objects []struct {
				Profession      string `json:"profession"`
				FirmName        string `json:"firm_name"`
				Link            string `json:"link"`
				VacancyRichText string `json:"vacancyRichText"`
				Candidat        string `json:"candidat"`
			} `json:"objects"`
			More bool `json:"more"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil {
			return err
		}
		if len(parsed.Objects) == 0 {
			return nil
		}
		for _, obj := range parsed.Objects {
			if obj.Link == "" {
				continue
			}
			emit(provider.Listing{Vacancy: provider.Vacancy{
				URL:         obj.Link,
				Title:       htmlutil.Text(obj.Profession),
				Company:     htmlutil.Text(obj.FirmName),
				Source:      "superjob",
				Description: htmlutil.Text(obj.VacancyRichText + " " + obj.Candidat),
			}})
		}
		if !parsed.More {
			return nil
		}
	}
	return nil
}

func (p *Provider) searchHTML(ctx context.Context, hint string, maxPages int, emit func(provider.Listing)) error {
	seen := map[string]struct{}{}
	limit := maxPages
	if limit == 0 {
		limit = 20
	}
	for page := 1; page <= limit; page++ {
		u, err := url.Parse(p.site() + "/vacancy/search/")
		if err != nil {
			return err
		}
		q := u.Query()
		q.Set("keywords", hint)
		if page > 1 {
			q.Set("page", fmt.Sprint(page))
		}
		u.RawQuery = q.Encode()
		body, err := p.HTTP.Get(ctx, u.String(), nil)
		if err != nil {
			return err
		}
		found := parseHTML(string(body))
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
	if job, ok := htmlutil.JobPosting(body); ok {
		if job.Title != "" {
			v.Title = job.Title
		}
		if job.Company != "" {
			v.Company = job.Company
		}
		if job.Description != "" {
			v.Description = job.Description
		}
		if job.URL != "" {
			v.URL = job.URL
		}
		if v.Posted.IsZero() {
			v.Posted = provider.ParseTime(job.Date)
		}
		if v.Location == "" {
			v.Location = job.Location
		}
	}
	return v, nil
}

func parseHTML(page string) []provider.Vacancy {
	matches := vacancyHref.FindAllString(page, -1)
	seen := map[string]struct{}{}
	var out []provider.Vacancy
	for _, href := range matches {
		href = htmlUnescape(href)
		if _, ok := seen[href]; ok {
			continue
		}
		seen[href] = struct{}{}
		title := htmlutil.AnchorText(page, href)
		out = append(out, provider.Vacancy{
			URL:    href,
			Title:  title,
			Source: "superjob",
		})
	}
	return out
}

func htmlUnescape(s string) string {
	return strings.NewReplacer("&amp;", "&", "&#38;", "&").Replace(s)
}
