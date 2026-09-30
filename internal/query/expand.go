package query

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Expand replaces alias names with their bodies, wrapped in parentheses.
// A quoted phrase is left as written. Cycles are an error.
func Expand(input string, aliases map[string]string) (string, error) {
	if strings.TrimSpace(input) == "" || len(aliases) == 0 {
		return input, nil
	}
	return expand(input, aliases, nil)
}

func expand(input string, aliases map[string]string, stack []string) (string, error) {
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
		case strings.HasPrefix(s[i:], "||"):
			b.WriteString("||")
			i += 2
		case s[i] == '&' || s[i] == '|' || s[i] == '(' || s[i] == ')':
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
				if strings.HasPrefix(s[j:], "&|") || strings.HasPrefix(s[j:], "||") || s[j] == '&' || s[j] == '|' || s[j] == '(' || s[j] == ')' || s[j] == '"' {
					break
				}
				rr, sz := utf8.DecodeRuneInString(s[j:])
				if unicode.IsSpace(rr) {
					break
				}
				j += sz
			}
			term := s[i:j]
			body, ok := aliases[term]
			if !ok {
				b.WriteString(term)
				i = j
				continue
			}
			for _, name := range stack {
				if name == term {
					return "", fmt.Errorf("alias cycle involving %s", term)
				}
			}
			next := append(append([]string{}, stack...), term)
			inner, err := expand(body, aliases, next)
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
