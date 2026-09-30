package hh

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const defaultBase = "https://api.hh.ru"

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider searches the public HeadHunter vacancies API.
type Provider struct {
	HTTP  Getter
	Base  string
	Token string
}

func (p *Provider) Name() string { return "hh" }

func (p *Provider) base() string {
	if p.Base != "" {
		return strings.TrimRight(p.Base, "/")
	}
	return defaultBase
}

func (p *Provider) Search(ctx context.Context, hints []string, maxPages int, emit func(provider.Listing)) error {
	if len(hints) == 0 {
		hints = []string{""}
	}
	var first error
	for _, hint := range hints {
		if err := p.searchHint(ctx, hint, maxPages, emit); err != nil && first == nil {
			first = err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return first
}

func (p *Provider) searchHint(ctx context.Context, hint string, maxPages int, emit func(provider.Listing)) error {
	for page := 0; maxPages == 0 || page < maxPages; page++ {
		u, err := url.Parse(p.base() + "/vacancies")
		if err != nil {
			return err
		}
		q := u.Query()
		q.Set("text", hint)
		q.Set("per_page", "100")
		q.Set("page", fmt.Sprint(page))
		u.RawQuery = q.Encode()
		body, err := p.HTTP.Get(ctx, u.String(), p.headers())
		if err != nil {
			return fmt.Errorf("search: %w", err)
		}
		parsed, err := parseSearch(body)
		if err != nil {
			return err
		}
		if len(parsed.Items) == 0 {
			return nil
		}
		for _, item := range parsed.Items {
			item := item
			v := item.vacancy()
			id := item.ID
			emit(provider.Listing{
				Vacancy: v,
				Fetch: func(ctx context.Context) (provider.Vacancy, error) {
					return p.detail(ctx, id, v)
				},
			})
		}
		if page+1 >= parsed.Pages {
			return nil
		}
	}
	return nil
}

func (p *Provider) detail(ctx context.Context, id string, fallback provider.Vacancy) (provider.Vacancy, error) {
	body, err := p.HTTP.Get(ctx, p.base()+"/vacancies/"+url.PathEscape(id), p.headers())
	if err != nil {
		return provider.Vacancy{}, err
	}
	var d struct {
		Name         string `json:"name"`
		AlternateURL string `json:"alternate_url"`
		Description  string `json:"description"`
		KeySkills    []struct {
			Name string `json:"name"`
		} `json:"key_skills"`
		Employer struct {
			Name string `json:"name"`
		} `json:"employer"`
	}
	if err := json.Unmarshal(body, &d); err != nil {
		return provider.Vacancy{}, err
	}
	if d.Name != "" {
		fallback.Title = htmlutil.Text(d.Name)
	}
	if d.AlternateURL != "" {
		fallback.URL = d.AlternateURL
	}
	if d.Employer.Name != "" {
		fallback.Company = htmlutil.Text(d.Employer.Name)
	}
	fallback.Description = htmlutil.Text(d.Description)
	fallback.Skills = nil
	for _, s := range d.KeySkills {
		if s.Name != "" {
			fallback.Skills = append(fallback.Skills, s.Name)
		}
	}
	return fallback, nil
}

func (p *Provider) headers() map[string]string {
	h := map[string]string{"HH-User-Agent": "vscan/0.1 (personal vacancy scanner)"}
	if p.Token != "" {
		h["Authorization"] = "Bearer " + p.Token
	}
	return h
}

type searchPage struct {
	Items []item `json:"items"`
	Pages int    `json:"pages"`
}

type item struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	AlternateURL string `json:"alternate_url"`
	Employer     struct {
		Name string `json:"name"`
	} `json:"employer"`
	Snippet struct {
		Requirement    string `json:"requirement"`
		Responsibility string `json:"responsibility"`
	} `json:"snippet"`
	Published string `json:"published_at"`
	Schedule  struct {
		ID string `json:"id"`
	} `json:"schedule"`
	Area struct {
		Name string `json:"name"`
	} `json:"area"`
	Salary *struct {
		From     *float64 `json:"from"`
		To       *float64 `json:"to"`
		Currency string   `json:"currency"`
	} `json:"salary"`
}

func (it item) vacancy() provider.Vacancy {
	remote := ""
	if it.Schedule.ID == "remote" {
		remote = "yes"
	}
	salary := ""
	if it.Salary != nil {
		salary = provider.FormatMoney(it.Salary.From, it.Salary.To, it.Salary.Currency, "")
	}
	return provider.Vacancy{
		URL:      it.AlternateURL,
		Title:    htmlutil.Text(it.Name),
		Company:  htmlutil.Text(it.Employer.Name),
		Source:   "hh",
		Remote:   remote,
		Location: htmlutil.Text(it.Area.Name),
		Salary:   salary,
		Posted:   provider.ParseTime(it.Published),
		Description: htmlutil.Text(strings.TrimSpace(
			it.Snippet.Requirement + " " + it.Snippet.Responsibility,
		)),
	}
}

func parseSearch(body []byte) (searchPage, error) {
	var raw struct {
		Items []struct {
			ID           flexID `json:"id"`
			Name         string `json:"name"`
			AlternateURL string `json:"alternate_url"`
			Employer     struct {
				Name string `json:"name"`
			} `json:"employer"`
			Snippet struct {
				Requirement    string `json:"requirement"`
				Responsibility string `json:"responsibility"`
			} `json:"snippet"`
			Published string `json:"published_at"`
			Schedule  struct {
				ID string `json:"id"`
			} `json:"schedule"`
			Area struct {
				Name string `json:"name"`
			} `json:"area"`
			Salary *struct {
				From     *float64 `json:"from"`
				To       *float64 `json:"to"`
				Currency string   `json:"currency"`
			} `json:"salary"`
		} `json:"items"`
		Pages int `json:"pages"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return searchPage{}, err
	}
	out := searchPage{Pages: raw.Pages}
	for _, it := range raw.Items {
		out.Items = append(out.Items, item{
			ID:           string(it.ID),
			Name:         it.Name,
			AlternateURL: it.AlternateURL,
			Employer:     it.Employer,
			Snippet:      it.Snippet,
			Published:    it.Published,
			Schedule:     it.Schedule,
			Area:         it.Area,
			Salary:       it.Salary,
		})
	}
	return out, nil
}

type flexID string

func (f *flexID) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = flexID(s)
		return nil
	}
	*f = flexID(strings.TrimSpace(string(b)))
	return nil
}
