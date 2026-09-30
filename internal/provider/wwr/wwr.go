package wwr

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const defaultURL = "https://weworkremotely.com/categories/remote-programming-jobs.rss"

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider reads the public We Work Remotely programming RSS feed.
type Provider struct {
	HTTP Getter
	URL  string
}

func (p *Provider) Name() string { return "wwr" }

func (p *Provider) Search(ctx context.Context, hints []string, maxPages int, emit func(provider.Listing)) error {
	_ = hints
	_ = maxPages
	endpoint := p.URL
	if endpoint == "" {
		endpoint = defaultURL
	}
	body, err := p.HTTP.Get(ctx, endpoint, nil)
	if err != nil {
		return err
	}
	items, err := parse(body)
	if err != nil {
		return err
	}
	for _, it := range items {
		emit(provider.Listing{Vacancy: it})
	}
	return nil
}

type rssDoc struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Items []struct {
			Title       string `xml:"title"`
			Description string `xml:"description"`
			Link        string `xml:"link"`
			Category    string `xml:"category"`
			Region      string `xml:"region"`
			PubDate     string `xml:"pubDate"`
		} `xml:"item"`
	} `xml:"channel"`
}

func parse(body []byte) ([]provider.Vacancy, error) {
	var doc rssDoc
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("rss: %w", err)
	}
	var out []provider.Vacancy
	for _, it := range doc.Channel.Items {
		if it.Link == "" || strings.Contains(it.Link, ".rss") {
			continue
		}
		title := htmlutil.Text(it.Title)
		company, role := splitTitle(title)
		out = append(out, provider.Vacancy{
			URL:         it.Link,
			Title:       role,
			Company:     company,
			Source:      "wwr",
			Description: htmlutil.Text(it.Description),
			Tags:        []string{"remote", it.Category, it.Region},
			Remote:      "yes",
			Location:    htmlutil.Text(it.Region),
			Posted:      provider.ParseTime(it.PubDate),
		})
	}
	return out, nil
}

func splitTitle(title string) (company, role string) {
	company, role, ok := strings.Cut(title, ":")
	if !ok {
		return "", strings.TrimSpace(title)
	}
	return strings.TrimSpace(company), strings.TrimSpace(role)
}
