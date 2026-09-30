package hn

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const defaultURL = "https://hacker-news.firebaseio.com/v0/jobstories.json"

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider reads the public Hacker News job stories.
// The list is one payload. Each id is then loaded as an item.
type Provider struct {
	HTTP Getter
	URL  string
}

func (p *Provider) Name() string { return "hn" }

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
	var ids []int
	if err := json.Unmarshal(body, &ids); err != nil {
		return err
	}
	base, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		itemURL := itemEndpoint(base, id)
		payload, err := p.HTTP.Get(ctx, itemURL, nil)
		if err != nil {
			return fmt.Errorf("item %d: %w", id, err)
		}
		job, ok, err := parseItem(payload)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if _, dup := seen[job.URL]; dup {
			continue
		}
		seen[job.URL] = struct{}{}
		emit(provider.Listing{Vacancy: job})
	}
	return nil
}

func itemEndpoint(base *url.URL, id int) string {
	u := *base
	u.Path = strings.TrimSuffix(u.Path, "jobstories.json") + "item/" + strconv.Itoa(id) + ".json"
	u.RawQuery = ""
	return u.String()
}

func parseItem(body []byte) (provider.Vacancy, bool, error) {
	if string(body) == "null" {
		return provider.Vacancy{}, false, nil
	}
	var it struct {
		ID      int    `json:"id"`
		Type    string `json:"type"`
		Title   string `json:"title"`
		Text    string `json:"text"`
		URL     string `json:"url"`
		Time    int64  `json:"time"`
		Dead    bool   `json:"dead"`
		Deleted bool   `json:"deleted"`
	}
	if err := json.Unmarshal(body, &it); err != nil {
		return provider.Vacancy{}, false, err
	}
	if it.Dead || it.Deleted || it.Type != "job" || it.Title == "" || it.ID <= 0 {
		return provider.Vacancy{}, false, nil
	}
	link := it.URL
	if link == "" {
		link = "https://news.ycombinator.com/item?id=" + strconv.Itoa(it.ID)
	}
	return provider.Vacancy{
		URL:         link,
		Title:       htmlutil.Text(it.Title),
		Source:      "hn",
		Description: htmlutil.Text(it.Text),
		Posted:      provider.UnixTime(it.Time),
	}, true, nil
}
