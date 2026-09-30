package habr

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const defaultBase = "https://career.habr.com"

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider reads the public Habr Career RSS feed and JSON-LD on the vacancy page.
type Provider struct {
	HTTP Getter
	Base string
}

func (p *Provider) Name() string { return "habr" }

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
	_ = maxPages // the public RSS feed is a single page
	var first error
	for _, hint := range hints {
		if err := p.searchHint(ctx, hint, emit); err != nil && first == nil {
			first = err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return first
}

func (p *Provider) searchHint(ctx context.Context, hint string, emit func(provider.Listing)) error {
	u, err := url.Parse(p.base() + "/vacancies/rss")
	if err != nil {
		return err
	}
	q := u.Query()
	q.Set("q", hint)
	u.RawQuery = q.Encode()
	body, err := p.HTTP.Get(ctx, u.String(), nil)
	if err != nil {
		return err
	}
	items, err := parseRSS(body)
	if err != nil {
		return err
	}
	for _, it := range items {
		it := it
		v := provider.Vacancy{
			URL:         it.Link,
			Title:       htmlutil.Text(it.Title),
			Company:     htmlutil.Text(it.Author),
			Source:      "habr",
			Description: htmlutil.Text(it.Description),
			Posted:      provider.ParseTime(it.PubDate),
		}
		emit(provider.Listing{
			Vacancy: v,
			Fetch: func(ctx context.Context) (provider.Vacancy, error) {
				return p.detail(ctx, v)
			},
		})
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
		if v.Posted.IsZero() {
			v.Posted = provider.ParseTime(job.Date)
		}
		if v.Location == "" {
			v.Location = job.Location
		}
	}
	return v, nil
}

type rssDoc struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	Author      string `xml:"author"`
	PubDate     string `xml:"pubDate"`
}

func parseRSS(body []byte) ([]rssItem, error) {
	var doc rssDoc
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("rss: %w", err)
	}
	var out []rssItem
	for _, it := range doc.Channel.Items {
		if it.Link == "" {
			continue
		}
		out = append(out, it)
	}
	return out, nil
}
