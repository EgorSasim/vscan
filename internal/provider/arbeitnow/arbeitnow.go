package arbeitnow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const defaultURL = "https://www.arbeitnow.com/api/job-board-api"

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider reads the public Arbeitnow job-board API.
// The API asks users to link back to Arbeitnow; vscan prints the job page
// and names the source in --help.
type Provider struct {
	HTTP Getter
	URL  string
}

func (p *Provider) Name() string { return "arbeitnow" }

func (p *Provider) Search(ctx context.Context, hints []string, maxPages int, emit func(provider.Listing)) error {
	_ = hints
	endpoint := p.URL
	if endpoint == "" {
		endpoint = defaultURL
	}
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
		q.Set("page", fmt.Sprint(page))
		u.RawQuery = q.Encode()
		body, err := p.HTTP.Get(ctx, u.String(), nil)
		if err != nil {
			return err
		}
		jobs, more, err := parse(body)
		if err != nil {
			return err
		}
		for _, job := range jobs {
			emit(provider.Listing{Vacancy: job})
		}
		if !more || len(jobs) == 0 {
			return nil
		}
	}
}

func parse(body []byte) ([]provider.Vacancy, bool, error) {
	var raw struct {
		Data []struct {
			Slug        string   `json:"slug"`
			CompanyName string   `json:"company_name"`
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Remote      bool     `json:"remote"`
			Location    string   `json:"location"`
			CreatedAt   int64    `json:"created_at"`
			Tags        []string `json:"tags"`
			JobTypes    []string `json:"job_types"`
		} `json:"data"`
		Links struct {
			Next *string `json:"next"`
		} `json:"links"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, false, err
	}
	var out []provider.Vacancy
	for _, it := range raw.Data {
		if it.Slug == "" || it.Title == "" {
			continue
		}
		remote := "no"
		if it.Remote {
			remote = "yes"
		}
		out = append(out, provider.Vacancy{
			URL:         "https://www.arbeitnow.com/view/" + it.Slug,
			Title:       htmlutil.Text(it.Title),
			Company:     htmlutil.Text(it.CompanyName),
			Source:      "arbeitnow",
			Description: htmlutil.Text(it.Description),
			Tags:        append(append([]string{}, it.Tags...), it.JobTypes...),
			Remote:      remote,
			Location:    htmlutil.Text(it.Location),
			Posted:      provider.UnixTime(it.CreatedAt),
		})
	}
	more := raw.Links.Next != nil && strings.TrimSpace(*raw.Links.Next) != ""
	return out, more, nil
}
