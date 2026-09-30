package jobspresso

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const defaultURL = "https://jobspresso.co/jobs/feed/"

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider reads the Jobspresso jobs RSS feed.
// An unlimited search stops after five pages.
type Provider struct {
	HTTP Getter
	URL  string
}

func (p *Provider) Name() string { return "jobspresso" }

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
		q.Set("paged", fmt.Sprint(page))
		u.RawQuery = q.Encode()
		body, err := p.HTTP.Get(ctx, u.String(), nil)
		if err != nil {
			return err
		}
		jobs, err := parse(body)
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
		if added == 0 {
			return nil
		}
	}
}

type rssDoc struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Creator     string `xml:"creator"`
			Description string `xml:"description"`
			Encoded     string `xml:"encoded"`
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
		link := strings.TrimSpace(it.Link)
		if link == "" || strings.Contains(link, "/feed") {
			continue
		}
		desc := it.Encoded
		if desc == "" {
			desc = it.Description
		}
		company, location := splitCreator(it.Creator)
		out = append(out, provider.Vacancy{
			URL:         link,
			Title:       htmlutil.Text(it.Title),
			Company:     company,
			Source:      "jobspresso",
			Description: htmlutil.Text(desc),
			Remote:      "yes",
			Location:    location,
			Posted:      provider.ParseTime(it.PubDate),
		})
	}
	return out, nil
}

func splitCreator(s string) (company, location string) {
	s = strings.ReplaceAll(s, "<br/>", "<br>")
	s = strings.ReplaceAll(s, "<br />", "<br>")
	company, loc, ok := strings.Cut(s, "<br>")
	if !ok {
		return htmlutil.Text(s), ""
	}
	return htmlutil.Text(company), strings.TrimLeft(htmlutil.Text(loc), "⚲ ")
}
