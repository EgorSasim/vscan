package landing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const defaultURL = "https://landing.jobs/api/v1/jobs.json"

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider reads the public Landing.jobs list.
// Pages are 50 jobs. An unlimited search stops after five pages.
type Provider struct {
	HTTP Getter
	URL  string
}

func (p *Provider) Name() string { return "landing" }

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
		q.Set("offset", fmt.Sprint((page-1)*50))
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
		if added == 0 || len(jobs) < 50 {
			return nil
		}
	}
}

func parse(body []byte) ([]provider.Vacancy, error) {
	var raw []struct {
		Title     string   `json:"title"`
		URL       string   `json:"url"`
		Remote    bool     `json:"remote"`
		Type      string   `json:"type"`
		Currency  string   `json:"currency_code"`
		Low       *float64 `json:"gross_salary_low"`
		High      *float64 `json:"gross_salary_high"`
		Published string   `json:"published_at"`
		Role      string   `json:"role_description"`
		Must      *string  `json:"main_requirements"`
		Nice      *string  `json:"nice_to_have"`
		Tags      []string `json:"tags"`
		Locations []struct {
			City    string `json:"city"`
			Country string `json:"country_code"`
		} `json:"locations"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	var out []provider.Vacancy
	for _, it := range raw {
		if it.URL == "" || it.Title == "" {
			continue
		}
		desc := it.Role
		if it.Must != nil {
			desc += "\n" + *it.Must
		}
		if it.Nice != nil {
			desc += "\n" + *it.Nice
		}
		remote := "no"
		if it.Remote {
			remote = "yes"
		}
		var tags []string
		if it.Type != "" {
			tags = append(tags, it.Type)
		}
		out = append(out, provider.Vacancy{
			URL:         it.URL,
			Title:       htmlutil.Text(it.Title),
			Company:     companyFromURL(it.URL),
			Source:      "landing",
			Description: htmlutil.Text(desc),
			Skills:      it.Tags,
			Tags:        tags,
			Remote:      remote,
			Location:    landingPlaces(it.Locations),
			Salary:      provider.FormatMoney(it.Low, it.High, it.Currency, ""),
			Posted:      provider.ParseTime(it.Published),
		})
	}
	return out, nil
}

func companyFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || !strings.EqualFold(parts[0], "at") {
		return ""
	}
	return strings.ReplaceAll(parts[1], "-", " ")
}

func landingPlaces(places []struct {
	City    string `json:"city"`
	Country string `json:"country_code"`
}) string {
	var parts []string
	for _, p := range places {
		switch {
		case p.City != "" && p.Country != "":
			parts = append(parts, p.City+", "+p.Country)
		case p.City != "":
			parts = append(parts, p.City)
		case p.Country != "":
			parts = append(parts, p.Country)
		}
	}
	return strings.Join(parts, "; ")
}
