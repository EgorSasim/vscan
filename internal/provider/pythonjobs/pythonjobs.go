package pythonjobs

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const defaultURL = "https://www.python.org/jobs/feed/rss/"

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider reads the public Python.org jobs RSS feed.
// The feed is one payload and has no publication date.
type Provider struct {
	HTTP Getter
	URL  string
}

func (p *Provider) Name() string { return "python" }

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
			Source:      "python",
			Description: desc,
		}
		if strings.Contains(strings.ToLower(desc), "remote") {
			job.Remote = "yes"
		}
		emit(provider.Listing{Vacancy: job})
	}
	return nil
}
