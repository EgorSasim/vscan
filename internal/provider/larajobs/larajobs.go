package larajobs

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const defaultURL = "https://larajobs.com/feed"

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider reads the public LaraJobs RSS feed.
// The feed is one payload. The description is often empty; title, company, and tags are the search text.
type Provider struct {
	HTTP Getter
	URL  string
}

func (p *Provider) Name() string { return "larajobs" }

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
				Title    string `xml:"title"`
				Link     string `xml:"link"`
				PubDate  string `xml:"pubDate"`
				Location string `xml:"location"`
				Kind     string `xml:"job_type"`
				Salary   string `xml:"salary"`
				Company  string `xml:"company"`
				Tags     string `xml:"tags"`
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
		var skills []string
		for _, tag := range strings.Split(it.Tags, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				skills = append(skills, tag)
			}
		}
		var tags []string
		if it.Kind != "" {
			tags = append(tags, it.Kind)
		}
		loc := htmlutil.Text(it.Location)
		job := provider.Vacancy{
			URL:      it.Link,
			Title:    htmlutil.Text(it.Title),
			Company:  htmlutil.Text(it.Company),
			Source:   "larajobs",
			Skills:   skills,
			Tags:     tags,
			Location: loc,
			Salary:   htmlutil.Text(it.Salary),
			Posted:   provider.ParseTime(it.PubDate),
		}
		if strings.Contains(strings.ToLower(loc), "remote") {
			job.Remote = "yes"
		}
		emit(provider.Listing{Vacancy: job})
	}
	return nil
}
