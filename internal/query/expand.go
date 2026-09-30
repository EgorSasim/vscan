package query

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Expand replaces alias names with their bodies, wrapped in parentheses.
// A quoted phrase is left as written. Cycles are an error.
// Language presets are not applied. Use ExpandPresets for a search query.
func Expand(input string, aliases map[string]string) (string, error) {
	return expand(input, aliases, false, nil)
}

// ExpandPresets expands aliases and then built-in language presets.
// A user alias of the same name wins. A preset body is not expanded again.
func ExpandPresets(input string, aliases map[string]string) (string, error) {
	return expand(input, aliases, true, nil)
}

func expand(input string, aliases map[string]string, presets bool, stack []string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return input, nil
	}
	var b strings.Builder
	s := input
	i := 0
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case unicode.IsSpace(r):
			b.WriteRune(r)
			i += size
		case strings.HasPrefix(s[i:], "&|"):
			b.WriteString("&|")
			i += 2
		case s[i] == '&' || s[i] == '|' || s[i] == '!' || s[i] == '(' || s[i] == ')':
			b.WriteByte(s[i])
			i++
		case s[i] == '"':
			j, err := skipQuoted(s, i)
			if err != nil {
				return "", err
			}
			b.WriteString(s[i:j])
			i = j
		default:
			j := i
			for j < len(s) {
				if strings.HasPrefix(s[j:], "&|") || s[j] == '&' || s[j] == '|' || s[j] == '!' || s[j] == '(' || s[j] == ')' || s[j] == '"' {
					break
				}
				rr, sz := utf8.DecodeRuneInString(s[j:])
				if unicode.IsSpace(rr) {
					break
				}
				j += sz
			}
			term := s[i:j]
			body, ok := aliasBody(term, aliases)
			if !ok && presets {
				if expr, hit := LookupPreset(term); hit {
					b.WriteByte('(')
					b.WriteString(expr)
					b.WriteByte(')')
					i = j
					continue
				}
			}
			if !ok {
				b.WriteString(term)
				i = j
				continue
			}
			key := strings.ToLower(term)
			for _, name := range stack {
				if name == key {
					return "", fmt.Errorf("alias cycle involving %s", term)
				}
			}
			next := append(append([]string{}, stack...), key)
			inner, err := expand(body, aliases, presets, next)
			if err != nil {
				return "", err
			}
			b.WriteByte('(')
			b.WriteString(inner)
			b.WriteByte(')')
			i = j
		}
	}
	return b.String(), nil
}

func aliasBody(term string, aliases map[string]string) (string, bool) {
	if len(aliases) == 0 {
		return "", false
	}
	if body, ok := aliases[term]; ok {
		return body, true
	}
	var found string
	n := 0
	for name, body := range aliases {
		if strings.EqualFold(name, term) {
			found = body
			n++
		}
	}
	if n == 1 {
		return found, true
	}
	return "", false
}

func skipQuoted(s string, i int) (int, error) {
	i++
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			i += 2
			continue
		}
		if s[i] == '"' {
			return i + 1, nil
		}
		i++
	}
	return 0, fmt.Errorf("unclosed quote")
}
