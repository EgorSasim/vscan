// Package htmlutil turns listing markup into plain text and JobPosting fields.
package htmlutil

import (
	"bytes"
	"encoding/json"
	"html"
	"regexp"
	"strings"
	"unicode"
)

var tagRe = regexp.MustCompile(`(?s)<[^>]*>`)

// Text unescapes HTML, drops tags, and collapses whitespace.
func Text(s string) string {
	s = html.UnescapeString(s)
	s = tagRe.ReplaceAllString(s, " ")
	return collapse(s)
}

func collapse(s string) string {
	var b strings.Builder
	space := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			space = b.Len() > 0
			continue
		}
		if space {
			b.WriteByte(' ')
			space = false
		}
		b.WriteRune(r)
	}
	return b.String()
}

type jobPosting struct {
	Title       string
	Description string
	Company     string
	URL         string
	Date        string
	Location    string
}

// JobPosting finds the first schema.org JobPosting in JSON-LD blocks.
func JobPosting(page []byte) (jobPosting, bool) {
	for _, raw := range ldJSON(page) {
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			continue
		}
		if job, ok := walk(v); ok {
			job.Description = Text(job.Description)
			job.Title = Text(job.Title)
			job.Company = Text(job.Company)
			return job, true
		}
	}
	return jobPosting{}, false
}

func walk(v any) (jobPosting, bool) {
	switch t := v.(type) {
	case []any:
		for _, el := range t {
			if job, ok := walk(el); ok {
				return job, true
			}
		}
	case map[string]any:
		if isJobPosting(t["@type"]) {
			job := jobPosting{
				Title:       asString(t["title"]),
				Description: asString(t["description"]),
				URL:         asString(t["url"]),
				Date:        asString(t["datePosted"]),
				Location:    place(t["jobLocation"]),
			}
			if org, ok := t["hiringOrganization"].(map[string]any); ok {
				job.Company = asString(org["name"])
			}
			if job.Title != "" || job.Description != "" {
				return job, true
			}
		}
		if g, ok := t["@graph"]; ok {
			return walk(g)
		}
	}
	return jobPosting{}, false
}

func isJobPosting(v any) bool {
	switch t := v.(type) {
	case string:
		return t == "JobPosting"
	case []any:
		for _, el := range t {
			if isJobPosting(el) {
				return true
			}
		}
	}
	return false
}

func place(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		if s := asString(t["name"]); s != "" {
			return s
		}
		if addr, ok := t["address"].(map[string]any); ok {
			if s := asString(addr["addressLocality"]); s != "" {
				return s
			}
		}
	}
	return ""
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func ldJSON(page []byte) [][]byte {
	lower := bytes.ToLower(page)
	var out [][]byte
	from := 0
	for {
		i := bytes.Index(lower[from:], []byte("application/ld+json"))
		if i < 0 {
			break
		}
		i += from
		gt := bytes.IndexByte(page[i:], '>')
		if gt < 0 {
			break
		}
		start := i + gt + 1
		endRel := bytes.Index(lower[start:], []byte("</script"))
		if endRel < 0 {
			break
		}
		raw := bytes.TrimSpace(page[start : start+endRel])
		if len(raw) > 0 {
			out = append(out, raw)
		}
		from = start + endRel
	}
	return out
}

// AnchorText returns visible text of the first <a> whose href equals href
// or ends with it (ignoring a query string).
func AnchorText(page, href string) string {
	lower := strings.ToLower(page)
	from := 0
	for from < len(page) {
		i := strings.Index(lower[from:], "href=")
		if i < 0 {
			return ""
		}
		i += from
		if i+6 > len(page) {
			return ""
		}
		quote := page[i+5]
		if quote != '"' && quote != '\'' {
			from = i + 5
			continue
		}
		valStart := i + 6
		valEnd := strings.IndexByte(page[valStart:], quote)
		if valEnd < 0 {
			return ""
		}
		valEnd += valStart
		val := html.UnescapeString(page[valStart:valEnd])
		tagEnd := strings.IndexByte(page[valEnd:], '>')
		if tagEnd < 0 {
			return ""
		}
		contentStart := valEnd + tagEnd + 1
		closeRel := strings.Index(lower[contentStart:], "</a>")
		if closeRel < 0 {
			return ""
		}
		if hrefMatch(val, href) {
			return Text(page[contentStart : contentStart+closeRel])
		}
		next := contentStart + closeRel + 4
		if next <= from {
			return ""
		}
		from = next
	}
	return ""
}

func hrefMatch(got, want string) bool {
	got = strings.Split(got, "?")[0]
	want = strings.Split(want, "?")[0]
	return got == want || strings.HasSuffix(got, want)
}

// ClassBlock returns the inner HTML of the first element whose class attribute contains className.
func ClassBlock(page, className string) string {
	lower := strings.ToLower(page)
	key := `class="`
	from := 0
	for {
		i := strings.Index(lower[from:], key)
		if i < 0 {
			return ""
		}
		i += from
		valStart := i + len(key)
		valEnd := strings.IndexByte(page[valStart:], '"')
		if valEnd < 0 {
			return ""
		}
		valEnd += valStart
		classVal := page[valStart:valEnd]
		if strings.Contains(classVal, className) {
			tagEnd := strings.IndexByte(page[valEnd:], '>')
			if tagEnd < 0 {
				return ""
			}
			return inner(page[valEnd+tagEnd+1:])
		}
		from = valEnd + 1
	}
}

func inner(s string) string {
	depth := 1
	lower := strings.ToLower(s)
	i := 0
	for i < len(s) {
		if strings.HasPrefix(lower[i:], "<div") {
			depth++
			if j := strings.IndexByte(s[i:], '>'); j >= 0 {
				i += j + 1
				continue
			}
		}
		if strings.HasPrefix(lower[i:], "</div") {
			depth--
			if depth == 0 {
				return s[:i]
			}
			if j := strings.IndexByte(s[i:], '>'); j >= 0 {
				i += j + 1
				continue
			}
		}
		i++
	}
	return s
}
