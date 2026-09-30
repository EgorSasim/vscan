package remotive

import (
	"context"
	"encoding/json"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const defaultURL = "https://remotive.com/api/remote-jobs"

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider reads the public Remotive remote-jobs API.
// Remotive asks aggregators to credit remotive.com.
type Provider struct {
	HTTP Getter
	URL  string
}

func (p *Provider) Name() string { return "remotive" }

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
	var raw struct {
		Jobs []struct {
			URL         string   `json:"url"`
			Title       string   `json:"title"`
			Company     string   `json:"company_name"`
			Description string   `json:"description"`
			Location    string   `json:"candidate_required_location"`
			Published   string   `json:"publication_date"`
			Salary      string   `json:"salary"`
			Tags        []string `json:"tags"`
			JobType     string   `json:"job_type"`
		} `json:"jobs"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	var out []provider.Vacancy
	for _, it := range raw.Jobs {
		if it.URL == "" || it.Title == "" {
			continue
		}
		tags := append([]string{}, it.Tags...)
		if it.JobType != "" {
			tags = append(tags, it.JobType)
		}
		out = append(out, provider.Vacancy{
			URL:         it.URL,
			Title:       htmlutil.Text(it.Title),
			Company:     htmlutil.Text(it.Company),
			Source:      "remotive",
			Description: htmlutil.Text(it.Description),
			Tags:        tags,
			Remote:      "yes",
			Location:    htmlutil.Text(it.Location),
			Salary:      htmlutil.Text(it.Salary),
			Posted:      provider.ParseTime(it.Published),
		})
	}
	return out, nil
}
