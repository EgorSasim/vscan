package nomads

import (
	"context"
	"encoding/json"
	"strings"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const defaultURL = "https://www.workingnomads.com/api/exposed_jobs/"

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider reads the public Working Nomads job feed.
type Provider struct {
	HTTP Getter
	URL  string
}

func (p *Provider) Name() string { return "nomads" }

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
	jobs, err := parse(body)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		emit(provider.Listing{Vacancy: job})
	}
	return nil
}

func parse(body []byte) ([]provider.Vacancy, error) {
	var raw []struct {
		URL         string `json:"url"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Company     string `json:"company_name"`
		Category    string `json:"category_name"`
		Tags        string `json:"tags"`
		Location    string `json:"location"`
		Published   string `json:"pub_date"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	var out []provider.Vacancy
	for _, it := range raw {
		if it.URL == "" || it.Title == "" {
			continue
		}
		var tags []string
		if it.Category != "" {
			tags = append(tags, it.Category)
		}
		for _, tag := range strings.Split(it.Tags, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
		out = append(out, provider.Vacancy{
			URL:         it.URL,
			Title:       htmlutil.Text(it.Title),
			Company:     htmlutil.Text(it.Company),
			Source:      "nomads",
			Description: htmlutil.Text(it.Description),
			Tags:        tags,
			Remote:      "yes",
			Location:    htmlutil.Text(it.Location),
			Posted:      provider.ParseTime(it.Published),
		})
	}
	return out, nil
}
