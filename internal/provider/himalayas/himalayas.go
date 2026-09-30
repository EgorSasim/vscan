package himalayas

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const defaultURL = "https://himalayas.app/jobs/api"

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider reads the public Himalayas jobs API.
// Pages follow nextCursor. An unlimited search stops after five pages.
type Provider struct {
	HTTP Getter
	URL  string
}

func (p *Provider) Name() string { return "himalayas" }

func (p *Provider) Search(ctx context.Context, hints []string, maxPages int, emit func(provider.Listing)) error {
	_ = hints
	endpoint := p.URL
	if endpoint == "" {
		endpoint = defaultURL
	}
	cursor := ""
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
		q.Set("limit", "20")
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		u.RawQuery = q.Encode()
		body, err := p.HTTP.Get(ctx, u.String(), nil)
		if err != nil {
			return err
		}
		jobs, next, err := parse(body)
		if err != nil {
			return err
		}
		for _, job := range jobs {
			emit(provider.Listing{Vacancy: job})
		}
		if next == "" || next == cursor || len(jobs) == 0 {
			return nil
		}
		cursor = next
	}
}

func parse(body []byte) ([]provider.Vacancy, string, error) {
	var raw struct {
		NextCursor string `json:"nextCursor"`
		Jobs       []struct {
			Title       string          `json:"title"`
			Company     string          `json:"companyName"`
			Description string          `json:"description"`
			Excerpt     string          `json:"excerpt"`
			GUID        string          `json:"guid"`
			PubDate     int64           `json:"pubDate"`
			MinSalary   *float64        `json:"minSalary"`
			MaxSalary   *float64        `json:"maxSalary"`
			Currency    nullString      `json:"currency"`
			Period      nullString      `json:"salaryPeriod"`
			Seniority   json.RawMessage `json:"seniority"`
			Places      json.RawMessage `json:"locationRestrictions"`
		} `json:"jobs"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, "", err
	}
	var out []provider.Vacancy
	for _, it := range raw.Jobs {
		if it.GUID == "" || it.Title == "" {
			continue
		}
		desc := it.Description
		if desc == "" {
			desc = it.Excerpt
		}
		out = append(out, provider.Vacancy{
			URL:         it.GUID,
			Title:       htmlutil.Text(it.Title),
			Company:     htmlutil.Text(it.Company),
			Source:      "himalayas",
			Description: htmlutil.Text(desc),
			Tags:        splitRaw(it.Seniority),
			Remote:      "yes",
			Location:    joinRaw(it.Places),
			Salary:      provider.FormatMoney(it.MinSalary, it.MaxSalary, string(it.Currency), string(it.Period)),
			Posted:      provider.UnixTime(it.PubDate),
		})
	}
	return out, raw.NextCursor, nil
}

type nullString string

func (n *nullString) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*n = ""
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	*n = nullString(s)
	return nil
}

func splitRaw(raw json.RawMessage) []string {
	joined := joinRaw(raw)
	if joined == "" {
		return nil
	}
	return strings.Split(joined, ", ")
}

func joinRaw(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var one string
	if json.Unmarshal(raw, &one) == nil {
		return one
	}
	var list []string
	if json.Unmarshal(raw, &list) == nil {
		return strings.Join(list, ", ")
	}
	return ""
}
