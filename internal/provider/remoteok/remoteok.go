package remoteok

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const defaultURL = "https://remoteok.com/api"

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider reads the public Remote OK JSON feed.
// The feed asks aggregators to link back to Remote OK; vscan prints only the
// job URL on stdout and mentions the source in --help.
type Provider struct {
	HTTP Getter
	URL  string
}

func (p *Provider) Name() string { return "remoteok" }

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
	var raw []map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	var out []provider.Vacancy
	for _, item := range raw {
		position, _ := item["position"].(string)
		link, _ := item["url"].(string)
		if position == "" || link == "" {
			continue
		}
		company, _ := item["company"].(string)
		desc, _ := item["description"].(string)
		location, _ := item["location"].(string)
		posted := timeFrom(item["date"])
		if posted.IsZero() {
			posted = timeFrom(item["epoch"])
		}
		min, _ := item["salary_min"].(float64)
		max, _ := item["salary_max"].(float64)
		var tags []string
		switch t := item["tags"].(type) {
		case []any:
			for _, el := range t {
				if s, ok := el.(string); ok {
					tags = append(tags, s)
				}
			}
		}
		out = append(out, provider.Vacancy{
			URL:         link,
			Title:       htmlutil.Text(position),
			Company:     htmlutil.Text(company),
			Source:      "remoteok",
			Description: htmlutil.Text(desc),
			Tags:        append(tags, "remote"),
			Remote:      "yes",
			Location:    htmlutil.Text(location),
			Salary:      provider.FormatSalary(int(min), int(max), "", ""),
			Posted:      posted,
		})
	}
	return out, nil
}

func timeFrom(v any) time.Time {
	switch t := v.(type) {
	case string:
		return provider.ParseTime(t)
	case float64:
		return provider.UnixTime(int64(t))
	case json.Number:
		n, err := strconv.ParseInt(t.String(), 10, 64)
		if err != nil {
			return time.Time{}
		}
		return provider.UnixTime(n)
	default:
		return time.Time{}
	}
}
