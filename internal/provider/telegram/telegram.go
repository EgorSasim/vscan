// Package telegram reads public channel previews at t.me/s/<name>.
// It is not wired into the site list. Delete this package and the
// telegram block in cmd/vscan to remove channel search.
package telegram

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"vscan/internal/htmlutil"
	"vscan/internal/provider"
)

const preview = "https://t.me/s/"

// Channels is the list of public channels to read.
// Names are without the @. An empty list searches nothing.
var Channels = []string{
	"it_remote",
	"remote_jobs_ru",
	"itjobsfeed",
	"devvacancy",
	"remoteit",
	"remoteok",
	"java_vacancies",
	"python_vacancies",
	"qa_vacancies",
	"devops_vacancies",
}

// Getter fetches a URL body.
type Getter interface {
	Get(ctx context.Context, rawURL string, header map[string]string) ([]byte, error)
}

// Provider reads the channels above.
// An unlimited search walks at most three preview pages per channel.
type Provider struct {
	HTTP     Getter
	Channels []string
}

func (p *Provider) Name() string { return "telegram" }

func (p *Provider) Search(ctx context.Context, hints []string, maxPages int, emit func(provider.Listing)) error {
	_ = hints
	names := p.Channels
	if names == nil {
		names = Channels
	}
	capPages := maxPages
	if capPages == 0 {
		capPages = 3
	}
	seen := map[string]struct{}{}
	var first error
	for _, name := range names {
		name = strings.TrimPrefix(strings.TrimSpace(name), "@")
		if name == "" {
			continue
		}
		if err := p.channel(ctx, name, capPages, seen, emit); err != nil && first == nil {
			first = fmt.Errorf("%s: %w", name, err)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
	return first
}

func (p *Provider) channel(ctx context.Context, name string, pages int, seen map[string]struct{}, emit func(provider.Listing)) error {
	before := 0
	for page := 0; page < pages; page++ {
		u := preview + url.PathEscape(name)
		if before > 0 {
			u += "?before=" + strconv.Itoa(before)
		}
		body, err := p.HTTP.Get(ctx, u, nil)
		if err != nil {
			return err
		}
		posts := parsePosts(string(body))
		if len(posts) == 0 {
			return nil
		}
		added := 0
		minID := 0
		for _, post := range posts {
			if minID == 0 || post.ID < minID {
				minID = post.ID
			}
			if _, ok := seen[post.URL]; ok {
				continue
			}
			seen[post.URL] = struct{}{}
			emit(provider.Listing{Vacancy: post.Vacancy})
			added++
		}
		if added == 0 || minID <= 1 || minID == before {
			return nil
		}
		before = minID
	}
	return nil
}

type post struct {
	provider.Vacancy
	ID int
}

var (
	postRe = regexp.MustCompile(`data-post="([^"/]+)/(\d+)"`)
	timeRe = regexp.MustCompile(`datetime="([^"]+)"`)
	textRe = regexp.MustCompile(`(?s)tgme_widget_message_text[^>]*>(.*?)</div>`)
)

func parsePosts(page string) []post {
	idxs := postRe.FindAllStringSubmatchIndex(page, -1)
	var out []post
	for i, loc := range idxs {
		end := len(page)
		if i+1 < len(idxs) {
			end = idxs[i+1][0]
		}
		chunk := page[loc[0]:end]
		id, err := strconv.Atoi(chunk[loc[4]-loc[0] : loc[5]-loc[0]])
		if err != nil || id <= 0 {
			continue
		}
		channel := chunk[loc[2]-loc[0] : loc[3]-loc[0]]
		text := ""
		if m := textRe.FindStringSubmatch(chunk); len(m) == 2 {
			text = htmlutil.Text(m[1])
		}
		if text == "" {
			continue
		}
		when := ""
		if m := timeRe.FindStringSubmatch(chunk); len(m) == 2 {
			when = m[1]
		}
		job := provider.Vacancy{
			URL:         "https://t.me/" + channel + "/" + strconv.Itoa(id),
			Title:       firstLine(text),
			Source:      "telegram",
			Description: text,
			Posted:      provider.ParseTime(when),
			Tags:        []string{channel},
		}
		if strings.Contains(strings.ToLower(text), "remote") || strings.Contains(foldYo(text), "удален") {
			job.Remote = "yes"
		}
		out = append(out, post{Vacancy: job, ID: id})
	}
	return out
}

func firstLine(text string) string {
	r := []rune(text)
	if len(r) > 140 {
		return string(r[:140])
	}
	return text
}

func foldYo(s string) string {
	s = strings.ToLower(s)
	return strings.ReplaceAll(s, "ё", "е")
}
