package nofluff

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const (
	defaultURL = "https://nofluffjobs.com/api/search/posting"
	searchBody = `{"criteriaSearch":{}}`
)

// Poster sends a JSON search body.
type Poster interface {
	Post(ctx context.Context, rawURL string, header map[string]string, body []byte) ([]byte, error)
}

// Provider reads the public NoFluffJobs search API.
// An unlimited search stops after five pages.
type Provider struct {
	HTTP Poster
	URL  string
}

func (p *Provider) Name() string { return "nofluff" }

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
		q.Set("salaryCurrency", "PLN")
		q.Set("salaryPeriod", "Month")
		q.Set("pageSize", "50")
		q.Set("pageTo", fmt.Sprint(page))
		if page > 1 {
			q.Set("pageFrom", fmt.Sprint(page))
		}
		u.RawQuery = q.Encode()
		body, err := p.HTTP.Post(ctx, u.String(), map[string]string{
			"Content-Type": "application/infiniteSearch+json",
			"Accept":       "application/json",
		}, []byte(searchBody))
		if err != nil {
			return err
		}
		jobs, pages, err := parse(body)
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
		if added == 0 || (pages > 0 && page >= pages) {
			return nil
		}
	}
}

func parse(body []byte) ([]provider.Vacancy, int, error) {
	var raw struct {
		TotalPages int `json:"totalPages"`
		Postings   []struct {
			Name        string          `json:"name"`
			Title       string          `json:"title"`
			URL         string          `json:"url"`
			Posted      int64           `json:"posted"`
			Seniority   []string        `json:"seniority"`
			Technology  json.RawMessage `json:"technology"`
			FullyRemote bool            `json:"fullyRemote"`
			Regions     []string        `json:"regions"`
			Location    struct {
				Places []struct {
					City    string `json:"city"`
					Country struct {
						Name string `json:"name"`
					} `json:"country"`
				} `json:"places"`
			} `json:"location"`
			Salary struct {
				From     *float64 `json:"from"`
				To       *float64 `json:"to"`
				Currency string   `json:"currency"`
				Period   string   `json:"period"`
				Type     string   `json:"type"`
			} `json:"salary"`
			Tiles struct {
				Values []struct {
					Value string `json:"value"`
				} `json:"values"`
			} `json:"tiles"`
		} `json:"postings"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, 0, err
	}
	var out []provider.Vacancy
	for _, it := range raw.Postings {
		slug := strings.TrimSpace(it.URL)
		if slug == "" || it.Title == "" {
			continue
		}
		var skills []string
		skills = append(skills, it.Seniority...)
		skills = append(skills, stringsFrom(it.Technology)...)
		for _, tile := range it.Tiles.Values {
			if tile.Value != "" {
				skills = append(skills, tile.Value)
			}
		}
		if it.Salary.Type != "" {
			skills = append(skills, it.Salary.Type)
		}
		remote := ""
		if it.FullyRemote {
			remote = "yes"
		}
		out = append(out, provider.Vacancy{
			URL:      "https://nofluffjobs.com/job/" + url.PathEscape(slug),
			Title:    htmlutil.Text(it.Title),
			Company:  htmlutil.Text(it.Name),
			Source:   "nofluff",
			Skills:   skills,
			Tags:     it.Regions,
			Remote:   remote,
			Location: placeList(it.Location.Places),
			Salary:   provider.FormatMoney(it.Salary.From, it.Salary.To, it.Salary.Currency, it.Salary.Period),
			Posted:   provider.UnixTime(it.Posted),
		})
	}
	return out, raw.TotalPages, nil
}

func placeList(places []struct {
	City    string `json:"city"`
	Country struct {
		Name string `json:"name"`
	} `json:"country"`
}) string {
	seen := map[string]struct{}{}
	var parts []string
	for _, p := range places {
		name := strings.TrimSpace(p.City)
		if name == "" {
			name = strings.TrimSpace(p.Country.Name)
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
	return strings.Join(parts, ", ")
}

func stringsFrom(raw json.RawMessage) []string {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var one string
	if err := json.Unmarshal(raw, &one); err == nil {
		if one == "" {
			return nil
		}
		return []string{one}
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		return list
	}
	var objs []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(raw, &objs); err == nil {
		var out []string
		for _, o := range objs {
			switch {
			case o.Name != "":
				out = append(out, o.Name)
			case o.Value != "":
				out = append(out, o.Value)
			}
		}
		return out
	}
	return nil
}
