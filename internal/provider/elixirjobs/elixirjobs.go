package elixirjobs

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const defaultURL = "https://elixirjobs.net/rss"

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider reads the public Elixir Jobs RSS feed.
// The feed is one payload.
type Provider struct {
	HTTP Getter
	URL  string
}

func (p *Provider) Name() string { return "elixir" }

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
	var doc struct {
		Channel struct {
			Items []struct {
				Title       string `xml:"title"`
				Link        string `xml:"link"`
				Description string `xml:"description"`
				PubDate     string `xml:"pubDate"`
			} `xml:"item"`
		} `xml:"channel"`
	}
	if err := xml.Unmarshal(body, &doc); err != nil {
		return fmt.Errorf("rss: %w", err)
	}
	for _, it := range doc.Channel.Items {
		if it.Link == "" || it.Title == "" {
			continue
		}
		desc := htmlutil.Text(it.Description)
		job := provider.Vacancy{
			URL:         it.Link,
			Title:       htmlutil.Text(it.Title),
			Source:      "elixir",
			Description: desc,
			Posted:      provider.ParseTime(it.PubDate),
		}
		if strings.Contains(fold(desc), "remote") {
			job.Remote = "yes"
		}
		emit(provider.Listing{Vacancy: job})
	}
	return nil
}

func fold(s string) string {
	return strings.ToLower(s)
}
