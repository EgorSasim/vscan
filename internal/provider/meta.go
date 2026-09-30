package provider

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DisplayRemote reports "yes", "no", or "" when it is unknown.
// An explicit board value wins. Otherwise a remote word in the title,
// tags, or description counts as yes. Absence does not mean no.
func (v Vacancy) DisplayRemote() string {
	switch strings.ToLower(strings.TrimSpace(v.Remote)) {
	case "yes", "true", "1", "remote":
		return "yes"
	case "no", "false", "0":
		return "no"
	}
	if looksRemote(v.Title) || looksRemote(strings.Join(v.Tags, " ")) || looksRemote(v.Description) {
		return "yes"
	}
	return ""
}

func looksRemote(s string) bool {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "ё", "е")
	for _, needle := range []string{"remote", "wfh", "work from home", "удален"} {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}

// ParseTime accepts RFC3339, RFC1123, a date, or a unix timestamp.
func ParseTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		time.RFC3339,
		time.RFC3339Nano,
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"2006-01-02",
		"2006-01-02T15:04:05",
		"02 Jan 2006 15:04:05 -0700",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return UnixTime(n)
	}
	return time.Time{}
}

// UnixTime accepts seconds or milliseconds.
func UnixTime(n int64) time.Time {
	if n <= 0 {
		return time.Time{}
	}
	if n > 1_000_000_000_000 {
		return time.UnixMilli(n).UTC()
	}
	return time.Unix(n, 0).UTC()
}

// Age is how long the vacancy has been up, as of now.
func Age(posted, now time.Time) string {
	if posted.IsZero() {
		return ""
	}
	d := now.Sub(posted)
	if d < 0 {
		d = 0
	}
	days := int(d.Hours() / 24)
	switch {
	case days < 1:
		return "today"
	case days == 1:
		return "1 day"
	default:
		return fmt.Sprintf("%d days", days)
	}
}

// FormatSalary renders a min-max range. Zero bounds are omitted.
func FormatSalary(min, max int, currency, period string) string {
	if min <= 0 && max <= 0 {
		return ""
	}
	var b strings.Builder
	switch {
	case min > 0 && max > 0 && min != max:
		fmt.Fprintf(&b, "%d-%d", min, max)
	case min > 0:
		fmt.Fprintf(&b, "%d", min)
	default:
		fmt.Fprintf(&b, "%d", max)
	}
	if currency != "" {
		b.WriteByte(' ')
		b.WriteString(currency)
	}
	if period != "" {
		b.WriteByte(' ')
		b.WriteString(period)
	}
	return b.String()
}

// FormatMoney is FormatSalary for nullable JSON numbers.
func FormatMoney(from, to *float64, currency, period string) string {
	min, max := 0, 0
	if from != nil {
		min = int(*from)
	}
	if to != nil {
		max = int(*to)
	}
	return FormatSalary(min, max, currency, period)
}
