package jobicy

import (
	"context"
	"encoding/json"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const defaultURL = "https://jobicy.com/api/v2/remote-jobs?count=100"

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider reads the public Jobicy remote-jobs API.
// Jobicy asks aggregators to credit jobicy.com.
type Provider struct {
	HTTP Getter
	URL  string
}

func (p *Provider) Name() string { return "jobicy" }

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
			Title       string   `json:"jobTitle"`
			Company     string   `json:"companyName"`
			Description string   `json:"jobDescription"`
			Excerpt     string   `json:"jobExcerpt"`
			Geo         string   `json:"jobGeo"`
			Level       string   `json:"jobLevel"`
			Type        string   `json:"jobType"`
			Published   string   `json:"pubDate"`
			SalaryMin   *float64 `json:"salaryMin"`
			SalaryMax   *float64 `json:"salaryMax"`
			Currency    string   `json:"salaryCurrency"`
			Period      string   `json:"salaryPeriod"`
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
		desc := it.Description
		if desc == "" {
			desc = it.Excerpt
		}
		var tags []string
		if it.Level != "" {
			tags = append(tags, it.Level)
		}
		if it.Type != "" {
			tags = append(tags, it.Type)
		}
		out = append(out, provider.Vacancy{
			URL:         it.URL,
			Title:       htmlutil.Text(it.Title),
			Company:     htmlutil.Text(it.Company),
			Source:      "jobicy",
			Description: htmlutil.Text(desc),
			Tags:        tags,
			Remote:      "yes",
			Location:    htmlutil.Text(it.Geo),
			Salary:      provider.FormatMoney(it.SalaryMin, it.SalaryMax, it.Currency, it.Period),
			Posted:      provider.ParseTime(it.Published),
		})
	}
	return out, nil
}
