// Package query parses the vacancy search language and matches it against text.
//
// Operators, longest match first: &|  ||  &
// & and &| bind tighter than ||. Parentheses group.
// The operator to the left of a word sets its mode. The first word takes the
// mode of the first operator.
//
//	&   literal whole word, case-insensitive, no synonyms
//	&|  smart: case-insensitive, ё=е, word or prefix, synonyms, light plurals
//	||  either branch
package query

import (
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Mode int

const (
	literal Mode = iota
	smart
)

// Expr is a parsed query.
type Expr interface {
	match(words []string, syn *Synonyms) bool
	hints() []string
	leaves() []string
}

type term struct {
	raw   string
	smart bool
	words []string
}

type andExpr struct {
	parts []Expr
}

type orExpr struct {
	left, right Expr
}

// Parse compiles a query string.
func Parse(input string) (Expr, error) {
	toks, err := lex(input)
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks}
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tokEOF {
		return nil, fmt.Errorf("trailing input near %q", p.peek().text)
	}
	if expr == nil {
		return nil, fmt.Errorf("empty query")
	}
	return expr, nil
}

// Match reports whether text satisfies the query.
func Match(e Expr, text string, syn *Synonyms) bool {
	if e == nil {
		return false
	}
	if syn == nil {
		syn = BuiltinSynonyms()
	}
	return e.match(words(text), syn)
}

// Hints are remote-search strings. The local matcher remains the source of truth.
// AND terms are sent together; OR becomes several hints. The list is capped.
func Hints(e Expr) []string {
	if e == nil {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	for _, h := range e.hints() {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		if _, ok := seen[h]; ok {
			continue
		}
		seen[h] = struct{}{}
		out = append(out, h)
		if len(out) == 8 {
			break
		}
	}
	return out
}

func (t *term) match(doc []string, syn *Synonyms) bool {
	if t.smart {
		return matchSmart(doc, t.words, syn.variants(t.raw))
	}
	return matchLiteral(doc, t.words)
}

func (t *term) hints() []string { return []string{t.raw} }

func (t *term) leaves() []string { return []string{t.raw} }

func (a *andExpr) match(doc []string, syn *Synonyms) bool {
	for _, p := range a.parts {
		if !p.match(doc, syn) {
			return false
		}
	}
	return true
}

func (a *andExpr) leaves() []string {
	var out []string
	for _, p := range a.parts {
		out = append(out, p.leaves()...)
	}
	return out
}

func (a *andExpr) hints() []string {
	if len(a.parts) == 0 {
		return nil
	}
	sets := make([][]string, len(a.parts))
	prod := 1
	for i, p := range a.parts {
		sets[i] = p.hints()
		if len(sets[i]) == 0 {
			sets[i] = []string{""}
		}
		prod *= len(sets[i])
		if prod > 8 {
			var flat []string
			for _, p := range a.parts {
				flat = append(flat, p.hints()...)
			}
			return flat
		}
	}
	cur := []string{""}
	for _, set := range sets {
		var next []string
		for _, prefix := range cur {
			for _, item := range set {
				joined := strings.TrimSpace(prefix + " " + item)
				next = append(next, joined)
			}
		}
		cur = next
	}
	return cur
}

func (o *orExpr) match(doc []string, syn *Synonyms) bool {
	return o.left.match(doc, syn) || o.right.match(doc, syn)
}

func (o *orExpr) hints() []string {
	return append(o.left.hints(), o.right.hints()...)
}

func (o *orExpr) leaves() []string {
	return append(o.left.leaves(), o.right.leaves()...)
}

// Terms returns every word or phrase in the expression, in source order.
func Terms(e Expr) []string {
	if e == nil {
		return nil
	}
	return e.leaves()
}

// CollectHints merges remote-search hints from several expressions, capped at 8.
func CollectHints(exprs ...Expr) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, e := range exprs {
		if e == nil {
			continue
		}
		for _, h := range Hints(e) {
			if _, ok := seen[h]; ok {
				continue
			}
			seen[h] = struct{}{}
			out = append(out, h)
			if len(out) == 8 {
				return out
			}
		}
	}
	return out
}

func matchLiteral(doc, phrase []string) bool {
	if len(phrase) == 0 {
		return false
	}
	return hasRun(doc, phrase, false)
}

func matchSmart(doc, phrase []string, variants [][]string) bool {
	if len(phrase) == 0 {
		return false
	}
	if hasRun(doc, phrase, true) {
		return true
	}
	for _, v := range variants {
		if sameWords(v, phrase) {
			continue
		}
		if hasRun(doc, v, true) {
			return true
		}
	}
	return false
}

func sameWords(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func hasRun(doc, phrase []string, smart bool) bool {
	if len(phrase) > len(doc) {
		return false
	}
	for i := 0; i+len(phrase) <= len(doc); i++ {
		ok := true
		for j := range phrase {
			if !wordMatch(doc[i+j], phrase[j], smart) {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func wordMatch(docWord, needle string, smart bool) bool {
	if docWord == needle {
		return true
	}
	if !smart {
		return false
	}
	for _, n := range pluralForms(needle) {
		if docWord == n {
			return true
		}
		if len([]rune(n)) >= 3 && strings.HasPrefix(docWord, n) {
			return true
		}
	}
	return false
}

func pluralForms(s string) []string {
	out := []string{s}
	if strings.Contains(s, " ") {
		return out
	}
	switch {
	case strings.HasSuffix(s, "es") && len(s) > 4:
		out = append(out, strings.TrimSuffix(s, "es"))
	case strings.HasSuffix(s, "s") && len(s) > 3 && !strings.HasSuffix(s, "ss"):
		out = append(out, strings.TrimSuffix(s, "s"))
	default:
		if len(s) > 2 {
			out = append(out, s+"s")
		}
	}
	return out
}

func words(s string) []string {
	s = normalize(s)
	var out []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return out
}

func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "ё", "е")
	return s
}

// Synonyms expands smart-mode terms. Groups are bidirectional.
type Synonyms struct {
	groups [][]string
	index  map[string]int
}

// BuiltinSynonyms is the default dictionary.
func BuiltinSynonyms() *Synonyms {
	s := &Synonyms{index: map[string]int{}}
	for _, g := range [][]string{
		{"remote", "relocate", "relocation", "удаленно", "удаленка", "wfh", "work from home", "remote work"},
		{"senior", "sr", "сеньор", "синьор", "старший"},
		{"middle", "mid", "мидл", "миддл"},
		{"junior", "jr", "джуниор", "младший"},
	} {
		s.add(g, true)
	}
	return s
}

// MergeFile overrides or adds groups from path.
// A line looks like: remote = relocate, wfh
// Missing file is not an error.
func (s *Synonyms) MergeFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for n, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, rest, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%s:%d: expected a line like remote = relocate, wfh", path, n+1)
		}
		parts := []string{key}
		for _, p := range strings.Split(rest, ",") {
			parts = append(parts, p)
		}
		s.add(parts, true)
	}
	return nil
}

func (s *Synonyms) add(parts []string, replace bool) {
	var norm []string
	seen := map[string]struct{}{}
	for _, p := range parts {
		w := words(p)
		if len(w) == 0 {
			continue
		}
		key := strings.Join(w, " ")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		norm = append(norm, key)
	}
	if len(norm) == 0 {
		return
	}
	idx := -1
	if replace {
		for _, n := range norm {
			if i, ok := s.index[n]; ok {
				idx = i
				break
			}
		}
	}
	if idx >= 0 {
		for _, old := range s.groups[idx] {
			delete(s.index, old)
		}
		s.groups[idx] = norm
	} else {
		idx = len(s.groups)
		s.groups = append(s.groups, norm)
	}
	for _, n := range norm {
		s.index[n] = idx
	}
}

// variants returns synonym phrases for raw, not including raw itself.
func (s *Synonyms) variants(raw string) [][]string {
	if s == nil {
		return nil
	}
	key := strings.Join(words(raw), " ")
	i, ok := s.index[key]
	if !ok {
		return nil
	}
	var out [][]string
	for _, alt := range s.groups[i] {
		if alt == key {
			continue
		}
		out = append(out, words(alt))
	}
	return out
}

type tokenKind int

const (
	tokEOF tokenKind = iota
	tokTerm
	tokAnd
	tokSmart
	tokOr
	tokLParen
	tokRParen
)

type token struct {
	kind tokenKind
	text string
}

func lex(input string) ([]token, error) {
	var toks []token
	s := input
	i := 0
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if unicode.IsSpace(r) {
			i += size
			continue
		}
		switch {
		case strings.HasPrefix(s[i:], "&|"):
			toks = append(toks, token{kind: tokSmart, text: "&|"})
			i += 2
		case strings.HasPrefix(s[i:], "||"):
			toks = append(toks, token{kind: tokOr, text: "||"})
			i += 2
		case s[i] == '&':
			toks = append(toks, token{kind: tokAnd, text: "&"})
			i++
		case s[i] == '(':
			toks = append(toks, token{kind: tokLParen, text: "("})
			i++
		case s[i] == ')':
			toks = append(toks, token{kind: tokRParen, text: ")"})
			i++
		case s[i] == '"':
			text, next, err := readQuoted(s, i)
			if err != nil {
				return nil, err
			}
			if strings.TrimSpace(text) == "" {
				return nil, fmt.Errorf("empty quoted phrase")
			}
			toks = append(toks, token{kind: tokTerm, text: text})
			i = next
		case s[i] == '|':
			return nil, fmt.Errorf("single \"|\"; write || for OR")
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
			if j == i {
				return nil, fmt.Errorf("unexpected character %q", s[i:i+size])
			}
			toks = append(toks, token{kind: tokTerm, text: s[i:j]})
			i = j
		}
	}
	toks = append(toks, token{kind: tokEOF})
	return toks, nil
}

func readQuoted(s string, i int) (string, int, error) {
	var b strings.Builder
	i++ // opening quote
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			b.WriteByte(s[i+1])
			i += 2
			continue
		}
		if s[i] == '"' {
			return b.String(), i + 1, nil
		}
		b.WriteByte(s[i])
		i++
	}
	return "", 0, fmt.Errorf("unclosed quote")
}

type parser struct {
	toks []token
	i    int
}

func (p *parser) peek() token {
	if p.i >= len(p.toks) {
		return token{kind: tokEOF}
	}
	return p.toks[p.i]
}

func (p *parser) next() token {
	t := p.peek()
	if p.i < len(p.toks) {
		p.i++
	}
	return t
}

func (p *parser) parseOr() (Expr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tokOr {
		p.next()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		if right == nil {
			return nil, fmt.Errorf("|| needs a word or a parenthesis")
		}
		left = &orExpr{left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseAnd() (Expr, error) {
	first, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	op := p.peek().kind
	if op != tokAnd && op != tokSmart {
		return first, nil
	}
	p.next()
	if first == nil {
		return nil, fmt.Errorf("operator needs a word before it")
	}
	parts := []Expr{tag(first, op == tokSmart)}
	for {
		item, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		if item == nil {
			return nil, fmt.Errorf("operator needs a word or a parenthesis after it")
		}
		parts = append(parts, tag(item, op == tokSmart))
		op = p.peek().kind
		if op != tokAnd && op != tokSmart {
			break
		}
		p.next()
	}
	return &andExpr{parts: parts}, nil
}

func (p *parser) parsePrimary() (Expr, error) {
	switch p.peek().kind {
	case tokLParen:
		p.next()
		inner, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if inner == nil {
			return nil, fmt.Errorf("empty parentheses")
		}
		if p.peek().kind != tokRParen {
			return nil, fmt.Errorf("unclosed parenthesis")
		}
		p.next()
		return inner, nil
	case tokTerm:
		t := p.next()
		ws := words(t.text)
		if len(ws) == 0 {
			return nil, fmt.Errorf("empty word")
		}
		return &term{raw: t.text, words: ws}, nil
	default:
		return nil, nil
	}
}

func tag(e Expr, smart bool) Expr {
	if t, ok := e.(*term); ok {
		cp := *t
		cp.smart = smart
		return &cp
	}
	return e
}
