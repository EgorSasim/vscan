package getmatch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const defaultBase = "https://getmatch.ru"

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider pages the public offers JSON. Filtering is local: the endpoint
// returns the open catalog, not a keyword search.
type Provider struct {
	HTTP Getter
	Base string
}

func (p *Provider) Name() string { return "getmatch" }

func (p *Provider) base() string {
	if p.Base != "" {
		return strings.TrimRight(p.Base, "/")
	}
	return defaultBase
}

func (p *Provider) Search(ctx context.Context, hints []string, maxPages int, emit func(provider.Listing)) error {
	_ = hints
	offset := 0
	const limit = 100
	pages := 0
	for {
		if maxPages > 0 && pages >= maxPages {
			return nil
		}
		u, err := url.Parse(p.base() + "/api/offers")
		if err != nil {
			return err
		}
		q := u.Query()
		q.Set("limit", fmt.Sprint(limit))
		q.Set("offset", fmt.Sprint(offset))
		u.RawQuery = q.Encode()
		body, err := p.HTTP.Get(ctx, u.String(), nil)
		if err != nil {
			return err
		}
		batch, total, rawCount, err := parse(body, p.base())
		if err != nil {
			return err
		}
		pages++
		if rawCount == 0 {
			return nil
		}
		for _, v := range batch {
			emit(provider.Listing{Vacancy: v})
		}
		offset += rawCount
		if total == 0 || offset >= total {
			return nil
		}
	}
}

func parse(body []byte, origin string) ([]provider.Vacancy, int, int, error) {
	var raw struct {
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
		Offers []struct {
			ID              int    `json:"id"`
			Position        string `json:"position"`
			URL             string `json:"url"`
			IsActive        bool   `json:"is_active"`
			Offer           string `json:"offer_description"`
			DescriptionHTML string `json:"description_html"`
			Company         struct {
				Name string `json:"name"`
			} `json:"company"`
			Locations []struct {
				Format string `json:"format"`
			} `json:"location_requirements"`
			Published string `json:"published_at"`
			Salary    *struct {
				From     *float64 `json:"from"`
				To       *float64 `json:"to"`
				Currency string   `json:"currency"`
			} `json:"salary"`
		} `json:"offers"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, 0, 0, err
	}
	var out []provider.Vacancy
	for _, o := range raw.Offers {
		if !o.IsActive || o.URL == "" {
			continue
		}
		var tags []string
		remote := ""
		for _, loc := range o.Locations {
			if loc.Format != "" {
				tags = append(tags, loc.Format)
			}
			if strings.EqualFold(loc.Format, "remote") {
				remote = "yes"
			}
		}
		salary := ""
		if o.Salary != nil {
			salary = provider.FormatMoney(o.Salary.From, o.Salary.To, o.Salary.Currency, "")
		}
		out = append(out, provider.Vacancy{
			URL:         provider.Abs(origin, o.URL),
			Title:       htmlutil.Text(o.Position),
			Company:     htmlutil.Text(o.Company.Name),
			Source:      "getmatch",
			Description: htmlutil.Text(o.Offer + " " + o.DescriptionHTML),
			Tags:        tags,
			Remote:      remote,
			Salary:      salary,
			Posted:      provider.ParseTime(o.Published),
		})
	}
	return out, raw.Meta.Total, len(raw.Offers), nil
}
