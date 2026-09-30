package trudvsem

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
	defaultURL = "https://opendata.trudvsem.ru/api/v1/vacancies"
	pageSize   = 100
)

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider reads the public open-data API of Работа в России.
// An unlimited search stops after five pages per hint.
type Provider struct {
	HTTP Getter
	URL  string
}

func (p *Provider) Name() string { return "trudvsem" }

func (p *Provider) Search(ctx context.Context, hints []string, maxPages int, emit func(provider.Listing)) error {
	if len(hints) == 0 {
		hints = []string{""}
	}
	endpoint := p.URL
	if endpoint == "" {
		endpoint = defaultURL
	}
	seen := map[string]struct{}{}
	var first error
	for _, hint := range hints {
		if err := p.searchHint(ctx, endpoint, hint, maxPages, seen, emit); err != nil && first == nil {
			first = err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return first
}

func (p *Provider) searchHint(ctx context.Context, endpoint, hint string, maxPages int, seen map[string]struct{}, emit func(provider.Listing)) error {
	for page := 0; ; page++ {
		if maxPages > 0 && page >= maxPages {
			return nil
		}
		if maxPages == 0 && page >= 5 {
			return nil
		}
		u, err := url.Parse(endpoint)
		if err != nil {
			return err
		}
		q := u.Query()
		if hint != "" {
			q.Set("text", hint)
		}
		q.Set("offset", fmt.Sprint(page*pageSize))
		q.Set("limit", fmt.Sprint(pageSize))
		u.RawQuery = q.Encode()
		body, err := p.HTTP.Get(ctx, u.String(), nil)
		if err != nil {
			return fmt.Errorf("search: %w", err)
		}
		jobs, total, err := parse(body)
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
		if added == 0 || len(jobs) < pageSize || (page+1)*pageSize >= total {
			return nil
		}
	}
}

func parse(body []byte) ([]provider.Vacancy, int, error) {
	var raw struct {
		Status string `json:"status"`
		Meta   struct {
			Total int `json:"total"`
		} `json:"meta"`
		Results struct {
			Vacancies []struct {
				Vacancy struct {
					Title       string   `json:"job-name"`
					URL         string   `json:"vac_url"`
					Created     string   `json:"creation-date"`
					SalaryMin   *float64 `json:"salary_min"`
					SalaryMax   *float64 `json:"salary_max"`
					Currency    string   `json:"currency"`
					Schedule    string   `json:"schedule"`
					Duty        string   `json:"duty"`
					Requirement flexNote `json:"requirement"`
					Needs       string   `json:"requirements"`
					Qualify     string   `json:"qualification"`
					Skills      flexText `json:"skills"`
					Company     struct {
						Name string `json:"name"`
					} `json:"company"`
					Region struct {
						Name string `json:"name"`
					} `json:"region"`
					Category struct {
						Name string `json:"specialisation"`
					} `json:"category"`
				} `json:"vacancy"`
			} `json:"vacancies"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, 0, err
	}
	if raw.Status != "" && raw.Status != "200" {
		return nil, 0, fmt.Errorf("status %s", raw.Status)
	}
	var out []provider.Vacancy
	for _, wrap := range raw.Results.Vacancies {
		it := wrap.Vacancy
		if it.URL == "" || it.Title == "" {
			continue
		}
		desc := it.Duty
		if it.Needs != "" {
			desc += "\n" + it.Needs
		}
		if it.Requirement.text != "" && it.Requirement.text != it.Needs {
			desc += "\n" + it.Requirement.text
		}
		var tags []string
		if it.Schedule != "" {
			tags = append(tags, it.Schedule)
		}
		if it.Category.Name != "" {
			tags = append(tags, it.Category.Name)
		}
		if it.Qualify != "" && !strings.EqualFold(strings.TrimSpace(it.Qualify), "не указано") {
			tags = append(tags, it.Qualify)
		}
		remote := ""
		if strings.Contains(fold(it.Schedule), "удален") {
			remote = "yes"
		}
		out = append(out, provider.Vacancy{
			URL:         it.URL,
			Title:       htmlutil.Text(it.Title),
			Company:     htmlutil.Text(it.Company.Name),
			Source:      "trudvsem",
			Description: htmlutil.Text(desc),
			Skills:      it.Skills,
			Tags:        tags,
			Remote:      remote,
			Location:    htmlutil.Text(it.Region.Name),
			Salary:      provider.FormatMoney(it.SalaryMin, it.SalaryMax, htmlutil.Text(it.Currency), ""),
			Posted:      provider.ParseTime(it.Created),
		})
	}
	return out, raw.Meta.Total, nil
}

func fold(s string) string {
	s = strings.ToLower(s)
	return strings.ReplaceAll(s, "ё", "е")
}

// flexNote accepts a string or the object Трудовой портал sends for requirement.
type flexNote struct {
	text string
}

func (f *flexNote) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	}
	var one string
	if err := json.Unmarshal(b, &one); err == nil {
		f.text = one
		return nil
	}
	var obj struct {
		Education  string          `json:"education"`
		Experience json.RawMessage `json:"experience"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return nil
	}
	var parts []string
	if obj.Education != "" {
		parts = append(parts, obj.Education)
	}
	exp := strings.Trim(string(obj.Experience), `"`)
	if exp != "" && exp != "null" {
		parts = append(parts, "опыт "+exp)
	}
	f.text = strings.Join(parts, ", ")
	return nil
}

// flexText accepts a string or an array of strings. Other shapes are ignored.
type flexText []string

func (f *flexText) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*f = nil
		return nil
	}
	var one string
	if err := json.Unmarshal(b, &one); err == nil {
		if one == "" {
			*f = nil
			return nil
		}
		*f = []string{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(b, &many); err == nil {
		*f = many
		return nil
	}
	*f = nil
	return nil
}
