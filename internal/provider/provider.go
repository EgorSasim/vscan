// Package provider is the vacancy source interface shared by every board.
package provider

import (
	"context"
	"net/url"
	"strings"
	"time"
)

// Vacancy is one opening. Description, skills and tags are all searched.
type Vacancy struct {
	URL         string
	Title       string
	Company     string
	Source      string
	Description string
	Skills      []string
	Tags        []string
	// Remote is "yes", "no", or empty when the board did not say.
	Remote   string
	Location string
	Salary   string
	Posted   time.Time
}

// Document is the text the query matcher sees.
func (v Vacancy) Document() string {
	var b strings.Builder
	b.WriteString(v.Title)
	b.WriteByte('\n')
	b.WriteString(v.Company)
	b.WriteByte('\n')
	b.WriteString(v.Description)
	b.WriteByte('\n')
	b.WriteString(strings.Join(v.Skills, " "))
	b.WriteByte('\n')
	b.WriteString(strings.Join(v.Tags, " "))
	return b.String()
}

// Listing is a search hit. Fetch loads the full description when the snippet
// is not enough for a match. Fetch may be nil.
type Listing struct {
	Vacancy Vacancy
	Fetch   func(context.Context) (Vacancy, error)
}

// Provider searches one board and calls emit as soon as each listing is known.
type Provider interface {
	Name() string
	Search(ctx context.Context, hints []string, maxPages int, emit func(Listing)) error
}

// Canonical drops fragments, tracking params and a trailing slash.
func Canonical(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return strings.TrimSpace(raw)
	}
	u.Fragment = ""
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	q := u.Query()
	for _, k := range []string{
		"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content",
		"from", "hhtmFrom",
	} {
		q.Del(k)
	}
	u.RawQuery = q.Encode()
	if len(u.Path) > 1 {
		u.Path = strings.TrimSuffix(u.Path, "/")
	}
	return u.String()
}

// Abs resolves ref against origin.
func Abs(origin, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	u, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	if u.IsAbs() {
		return u.String()
	}
	base, err := url.Parse(origin)
	if err != nil {
		return ref
	}
	return base.ResolveReference(u).String()
}
