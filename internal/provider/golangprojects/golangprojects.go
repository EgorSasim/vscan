package golangprojects

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const defaultURL = "https://www.golangprojects.com/rss.xml"

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider reads the public Golangprojects RSS feed.
// The feed is one payload. The title is "role @ company".
type Provider struct {
	HTTP Getter
	URL  string
}

func (p *Provider) Name() string { return "golangprojects" }

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
		title := htmlutil.Text(it.Title)
		role, company := splitTitle(title)
		desc := htmlutil.Text(it.Description)
		job := provider.Vacancy{
			URL:         it.Link,
			Title:       role,
			Company:     company,
			Source:      "golangprojects",
			Description: desc,
		}
		if strings.Contains(strings.ToLower(desc), "remote") {
			job.Remote = "yes"
		}
		emit(provider.Listing{Vacancy: job})
	}
	return nil
}

func splitTitle(title string) (role, company string) {
	title = strings.ReplaceAll(title, "\u00a0", " ")
	role, company, ok := strings.Cut(title, " @ ")
	if !ok {
		return strings.TrimSpace(title), ""
	}
	return strings.TrimSpace(role), strings.TrimSpace(company)
}
