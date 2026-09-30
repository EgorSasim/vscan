package fourday

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const defaultURL = "https://4dayweek.io/api/jobs"

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider reads the public 4dayweek.io jobs API.
// An unlimited search stops after five pages.
type Provider struct {
	HTTP Getter
	URL  string
}

func (p *Provider) Name() string { return "fourday" }

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
		added := 0
		for _, job := range jobs {
			if _, ok := seen[job.URL]; ok {
				continue
			}
			seen[job.URL] = struct{}{}
			emit(provider.Listing{Vacancy: job})
			added++
		}
		if added == 0 || !more {
			return nil
		}
	}
}

func parse(body []byte) ([]provider.Vacancy, bool, error) {
	var raw struct {
		HasMore bool `json:"has_more"`
		Jobs    []struct {
			Title       string `json:"title"`
			Slug        string `json:"slug"`
			Company     string `json:"company_name"`
			Arrangement string `json:"work_arrangement"`
			Posted      int64  `json:"posted"`
			Category    string `json:"category"`
			Level       string `json:"level"`
			Schedule    string `json:"schedule_type"`
			Expired     bool   `json:"is_expired"`
			Stack       []struct {
				Name string `json:"name"`
			} `json:"stack"`
			Locations []struct {
				City        string `json:"city"`
				Country     string `json:"country"`
				Arrangement string `json:"work_arrangement"`
			} `json:"locations"`
		} `json:"jobs"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, false, err
	}
	var out []provider.Vacancy
	for _, it := range raw.Jobs {
		if it.Expired || it.Slug == "" || it.Title == "" {
			continue
		}
		var skills []string
		for _, s := range it.Stack {
			if s.Name != "" {
				skills = append(skills, s.Name)
			}
		}
		if it.Level != "" {
			skills = append(skills, it.Level)
		}
		var tags []string
		if it.Category != "" {
			tags = append(tags, it.Category)
		}
		if it.Schedule != "" {
			tags = append(tags, it.Schedule)
		}
		out = append(out, provider.Vacancy{
			URL:      "https://4dayweek.io/job/" + url.PathEscape(it.Slug),
			Title:    htmlutil.Text(it.Title),
			Company:  htmlutil.Text(it.Company),
			Source:   "fourday",
			Skills:   skills,
			Tags:     tags,
			Remote:   arrangement(it.Arrangement),
			Location: fourPlaces(it.Locations),
			Posted:   provider.UnixTime(it.Posted),
		})
	}
	return out, raw.HasMore, nil
}

func arrangement(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "remote":
		return "yes"
	case "onsite", "on-site", "office", "in-office":
		return "no"
	default:
		return ""
	}
}

func fourPlaces(places []struct {
	City        string `json:"city"`
	Country     string `json:"country"`
	Arrangement string `json:"work_arrangement"`
}) string {
	seen := map[string]struct{}{}
	var parts []string
	for _, p := range places {
		name := strings.TrimSpace(p.City)
		if name == "" {
			name = strings.TrimSpace(p.Country)
		} else if p.Country != "" {
			name = name + ", " + strings.TrimSpace(p.Country)
		}
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		parts = append(parts, name)
	}
	return strings.Join(parts, "; ")
}
