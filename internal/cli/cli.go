// Package cli parses argv for the vscan command.
package cli

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Options is a parsed invocation. Query may still be empty when it comes from stdin.
type Options struct {
	Help          bool
	JSON          bool
	Verbose       bool
	Scanner       bool
	ClearCache    bool
	NoHistory     bool
	History       bool
	HistoryN      int
	Every         time.Duration
	EverySet      bool
	MaxPages      int
	Listen        string
	Webhook       string
	Sources       []string
	Query         string
	Company       string
	Profession    string
	Platform      string
	AliasList     bool
	AliasName     string
	AliasValue    string
	AliasDel      string
	Presets       bool
	Response      bool
	ResponseArg   string
	Headed        bool
	CredsAction   string
	CredsPlatform string
}

// Parse reads arguments after the program name.
func Parse(args []string) (Options, error) {
	if len(args) > 0 && args[0] == "alias" {
		return parseAlias(args[1:])
	}
	if len(args) > 0 && args[0] == "presets" {
		if len(args) != 1 {
			return Options{}, fmt.Errorf("vscan presets")
		}
		return Options{Presets: true}, nil
	}
	if len(args) > 0 && args[0] == "creds" {
		return parseCreds(args[1:])
	}
	opt := Options{MaxPages: 0, Every: 60 * time.Minute}
	var positionals []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-h" || a == "--help":
			opt.Help = true
			return opt, nil
		case a == "--json":
			opt.JSON = true
		case a == "--verbose" || a == "-v":
			opt.Verbose = true
		case a == "--scanner" || a == "--scan":
			opt.Scanner = true
		case a == "--clear-cache":
			opt.ClearCache = true
		case a == "--headed":
			opt.Headed = true
		case a == "--response" || strings.HasPrefix(a, "--response="):
			opt.Response = true
			if v, ok := strings.CutPrefix(a, "--response="); ok {
				if v == "" {
					return Options{}, fmt.Errorf("--response: pass URLs or omit the value to read stdin")
				}
				opt.ResponseArg = v
			} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				opt.ResponseArg = args[i]
			}
		case a == "--no-history":
			opt.NoHistory = true
		case a == "--history" || strings.HasPrefix(a, "--history="):
			opt.History = true
			raw := ""
			if v, ok := strings.CutPrefix(a, "--history="); ok {
				raw = v
			} else if i+1 < len(args) && isInt(args[i+1]) {
				i++
				raw = args[i]
			}
			if raw != "" {
				n, err := strconv.Atoi(raw)
				if err != nil || n < 1 {
					return Options{}, fmt.Errorf("--history: entry numbers start at 1")
				}
				opt.HistoryN = n
			}
		case a == "--listen" || strings.HasPrefix(a, "--listen="):
			opt.Listen = "127.0.0.1:8787"
			if v, ok := strings.CutPrefix(a, "--listen="); ok {
				if v != "" {
					opt.Listen = v
				}
			} else if i+1 < len(args) && looksLikeAddr(args[i+1]) {
				i++
				opt.Listen = args[i]
			}
		case a == "--webhook" || strings.HasPrefix(a, "--webhook="):
			v, err := takeValue(args, &i, a, "--webhook")
			if err != nil {
				return Options{}, err
			}
			opt.Webhook = v
		case a == "--every" || strings.HasPrefix(a, "--every="):
			v, err := takeValue(args, &i, a, "--every")
			if err != nil {
				return Options{}, err
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 {
				return Options{}, fmt.Errorf("--every: want a whole number of minutes, at least 1")
			}
			opt.Every = time.Duration(n) * time.Minute
			opt.EverySet = true
		case a == "--max-pages" || strings.HasPrefix(a, "--max-pages="):
			v, err := takeValue(args, &i, a, "--max-pages")
			if err != nil {
				return Options{}, err
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				return Options{}, fmt.Errorf("--max-pages: want a whole number, 0 means until the board stops")
			}
			opt.MaxPages = n
		case a == "--sources" || strings.HasPrefix(a, "--sources="):
			v, err := takeValue(args, &i, a, "--sources")
			if err != nil {
				return Options{}, err
			}
			opt.Sources = splitSources(v)
			if len(opt.Sources) == 0 {
				return Options{}, fmt.Errorf("--sources: list is empty")
			}
		case a == "--company" || strings.HasPrefix(a, "--company="):
			v, err := takeValue(args, &i, a, "--company")
			if err != nil {
				return Options{}, err
			}
			opt.Company = v
		case a == "--profession" || strings.HasPrefix(a, "--profession="):
			v, err := takeValue(args, &i, a, "--profession")
			if err != nil {
				return Options{}, err
			}
			opt.Profession = v
		case a == "--platform" || strings.HasPrefix(a, "--platform="):
			v, err := takeValue(args, &i, a, "--platform")
			if err != nil {
				return Options{}, err
			}
			opt.Platform = v
		case a == "--":
			positionals = append(positionals, args[i+1:]...)
			i = len(args)
		case strings.HasPrefix(a, "-"):
			return Options{}, fmt.Errorf("unknown flag %s (see --help)", a)
		default:
			positionals = append(positionals, a)
		}
	}
	if len(positionals) > 1 {
		return Options{}, fmt.Errorf("the query must be one argument; quote it")
	}
	if len(positionals) == 1 {
		opt.Query = positionals[0]
	}
	if opt.Help {
		return opt, nil
	}
	if opt.EverySet && !opt.Scanner {
		return Options{}, fmt.Errorf("--every only works with --scanner")
	}
	if opt.Listen != "" && !opt.Scanner {
		return Options{}, fmt.Errorf("--listen only works with --scanner")
	}
	if opt.Webhook != "" && !opt.Scanner {
		return Options{}, fmt.Errorf("--webhook only works with --scanner")
	}
	if opt.Listen != "" {
		if err := checkListen(opt.Listen); err != nil {
			return Options{}, err
		}
	}
	if opt.History && opt.HistoryN > 0 && opt.HasSearch() {
		return Options{}, fmt.Errorf("do not pass a search and --history N together")
	}
	if opt.History && opt.HistoryN == 0 && opt.HasSearch() {
		return Options{}, fmt.Errorf("--history without a number lists queries and does not search")
	}
	if opt.Headed && !opt.Response {
		return Options{}, fmt.Errorf("--headed only works with --response")
	}
	if opt.Response && (opt.HasSearch() || opt.Scanner || opt.History) {
		return Options{}, fmt.Errorf("--response does not run a search")
	}
	return opt, nil
}

// HasSearch reports whether a query or a field flag was given.
func (o Options) HasSearch() bool {
	return o.Query != "" || o.Company != "" || o.Profession != "" || o.Platform != ""
}

// SearchRecord is the search half of an invocation, suitable for history.
func (o Options) SearchRecord() string {
	var parts []string
	if o.Company != "" {
		parts = append(parts, "--company="+quoteArg(o.Company))
	}
	if o.Profession != "" {
		parts = append(parts, "--profession="+quoteArg(o.Profession))
	}
	if o.Platform != "" {
		parts = append(parts, "--platform="+quoteArg(o.Platform))
	}
	if o.Query != "" {
		parts = append(parts, quoteArg(o.Query))
	}
	return strings.Join(parts, " ")
}

func parseCreds(args []string) (Options, error) {
	if len(args) == 0 {
		return Options{CredsAction: "list"}, nil
	}
	switch args[0] {
	case "set", "delete":
		if len(args) != 2 {
			return Options{}, fmt.Errorf("vscan creds %s PLATFORM", args[0])
		}
		return Options{CredsAction: args[0], CredsPlatform: args[1]}, nil
	default:
		return Options{}, fmt.Errorf("vscan creds [set|delete] PLATFORM")
	}
}

func parseAlias(args []string) (Options, error) {
	opt := Options{}
	if len(args) == 0 {
		opt.AliasList = true
		return opt, nil
	}
	if args[0] == "-d" || args[0] == "--delete" {
		if len(args) != 2 {
			return Options{}, fmt.Errorf("vscan alias --delete NAME")
		}
		opt.AliasDel = args[1]
		return opt, nil
	}
	if strings.HasPrefix(args[0], "-") {
		return Options{}, fmt.Errorf("unknown flag %s (see --help)", args[0])
	}
	if len(args) > 2 {
		return Options{}, fmt.Errorf("vscan alias NAME=EXPRESSION")
	}
	name := args[0]
	body := ""
	if len(args) == 2 {
		body = args[1]
	} else if n, v, ok := strings.Cut(args[0], "="); ok {
		name, body = n, v
	}
	name = strings.TrimSpace(name)
	body = strings.TrimSpace(body)
	if body == "" {
		return Options{}, fmt.Errorf("vscan alias NAME=EXPRESSION")
	}
	opt.AliasName = name
	opt.AliasValue = body
	return opt, nil
}

// Split breaks a history line into arguments, honoring quotes.
func Split(s string) ([]string, error) {
	var args []string
	var b strings.Builder
	in := false
	quote := rune(0)
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if !in && (r == ' ' || r == '\t') {
			i += size
			continue
		}
		if !in {
			in = true
			b.Reset()
			quote = 0
		}
		if quote == 0 && (r == '\'' || r == '"') {
			quote = r
			i += size
			continue
		}
		if quote != 0 && r == '\\' && quote == '"' && i+size < len(s) {
			n, nsize := utf8.DecodeRuneInString(s[i+size:])
			b.WriteRune(n)
			i += size + nsize
			continue
		}
		if quote != 0 && r == quote {
			quote = 0
			i += size
			if i >= len(s) || s[i] == ' ' || s[i] == '\t' {
				args = append(args, b.String())
				in = false
			}
			continue
		}
		if quote == 0 && (r == ' ' || r == '\t') {
			args = append(args, b.String())
			in = false
			i += size
			continue
		}
		b.WriteRune(r)
		i += size
	}
	if quote != 0 {
		return nil, fmt.Errorf("unclosed quote in a saved query")
	}
	if in {
		args = append(args, b.String())
	}
	return args, nil
}

func quoteArg(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, " \t\"\\") {
		return s
	}
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

func splitSources(v string) []string {
	var out []string
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func takeValue(args []string, i *int, a, name string) (string, error) {
	if v, ok := strings.CutPrefix(a, name+"="); ok {
		if v == "" {
			return "", fmt.Errorf("%s: needs a value", name)
		}
		return v, nil
	}
	if *i+1 >= len(args) || strings.HasPrefix(args[*i+1], "-") {
		return "", fmt.Errorf("%s: needs a value", name)
	}
	*i++
	return args[*i], nil
}

func isInt(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func looksLikeAddr(s string) bool {
	if s == "" || strings.HasPrefix(s, "-") {
		return false
	}
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}
	_, port, err := net.SplitHostPort(s)
	if err != nil {
		return false
	}
	_, err = strconv.Atoi(port)
	return err == nil
}

func checkListen(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("--listen: want a host:port address")
	}
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("--listen accepts only loopback (127.0.0.1 or localhost)")
	}
	return nil
}

// Help is the --help text.
func Help() string {
	return `vscan — search programmer vacancies from the terminal

Usage:
  vscan [flags] [query]
  echo 'Senior&|Angular' | vscan
  vscan 'Senior&Angular' | wc -l

The query is one argument. Operators may have spaces around them:
  Senior&Angular             both words, whole word, no synonyms
  Senior||Angular            either word is enough
  Senior&|Angular&|Remote    smart AND: case-insensitive, a word or a
                             prefix (Angular matches AngularJS), synonyms
                             (remote matches relocate and wfh), and the
                             word may be in the title or the description
  Senior&Angular&|Remote     Senior and Angular are literal, Remote uses synonyms
  (Senior||Lead)&Angular     parentheses; & and &| bind tighter than ||
  "remote work"              a quoted phrase

One word with no operator is a literal search. Several words with no
operator are an error: join them with &, &|, or ||. A single | is an error.

Field flags use the same operators. Every flag that is set must match.
An alias name expands in place of a word, including inside flags.

  --company=TBank||AlfaBank
  --profession=Angular||AngularJS||TS||Typescript||JavaScript
  --platform=hh||habr

--company matches the company name only (T-Bank matches TBank).
--profession matches the title and skills, not the full description.
--platform picks boards. A vacancy has one source, so list several with ||.
Names: hh, habr, superjob, djinni, getmatch, geekjob, remoteok, wwr,
arbeitnow, remotive, jobicy, himalayas, nomads.
With --sources, the result is the intersection of the two lists.

Presets cover the usual spellings of a language or framework:
  vscan Go
  vscan Angular
  vscan 'C++'
  vscan presets

Go matches Golang, Angular matches AngularJS. A user alias with the
same name wins. Quote a name to keep that word only: "Go".

Aliases:
  vscan alias myStack='Angular||Typescript||JavaScript||JS||TS||React'
  vscan alias
  vscan alias --delete myStack

Alias file: ~/.config/vscan/aliases.

Auto-apply is omitted by -tags noresponse (make build, the VPS binary).
It is included only by -tags response (make build-response, bin/vscan-response).
If both tags are set, noresponse wins.
  vscan creds set hh
  vscan creds
  vscan creds delete hh
  vscan --response URLS
  vscan --response          read URLs from stdin, one per line

URLS may use newlines or the unit separator $'\x1f' between links.
Boards with an apply button: hh, habr, superjob, djinni, getmatch, geekjob.
Logins stay in the OS keychain (macOS) or Secret Service (Linux).
A captcha, a confirmation code, or extra form questions skip that vacancy.

Flags:
  -h, --help                 this help
  --json                     one JSON object per line instead of a bare URL
  --verbose, -v              human-readable block: remote, location, salary, age
  --company EXPR             company
  --profession EXPR          profession: title and skills
  --platform EXPR            boards from the list above
  --sources hh,habr,...      the same list, comma-separated, no operators
  --max-pages N              pages per board; 0 means until the board stops
  --scanner, --scan          repeat the search
  --every N                  minutes between passes (default 60)
  --listen [ADDR]            SSE on 127.0.0.1:8787 (GET /events, GET /health)
  --webhook URL              POST JSON for each new URL
  --history                  past queries, newest first
  --history N                run history entry N again
  --no-history               do not remember this query
  --clear-cache              forget remembered vacancy URLs, then exit
                             with --scanner, print the current matches again
  --response [URLS]          apply to vacancy URLs (optional build only)
  --headed                   show the browser window; only with --response

Default stdout is one URL per line. --json adds title, company, source,
and, when the board provides them, remote, location, salary, posted, age.
--verbose prints the same fields as a text block. Combine --json and
--verbose to keep JSON. Board errors go to stderr and the other boards
continue. Exit codes: 0 matches, a stopped scanner, or a closed pipe
(head); 1 nothing found; 2 bad arguments.

The scanner stores URLs it has already printed in
~/.cache/vscan/seen (or $XDG_CACHE_HOME/vscan/seen). When that file is
empty at start, the first pass only remembers the current results and
prints nothing. Later passes print a URL only when it is new.
--clear-cache deletes every remembered URL. Alone, it prints how many
links it removed and exits. Together with --scanner, the current matches
are printed once, then only new ones. A one-shot search (no --scanner)
does not use this cache. --listen and --webhook require --scanner and
can run together with stdout.

  vscan --scanner --every 30 --listen 127.0.0.1:8787 'Senior&|Angular'

A Telegram bot stays outside vscan. --scan is the same flag as --scanner:

  vscan --scanner --every 30 'Senior&|Angular' | while read -r url; do
    curl -s -X POST "https://api.telegram.org/bot$TOKEN/sendMessage" \
      -d chat_id="$CHAT" --data-urlencode text="$url"
  done

The first scanner pass is silent unless you also pass --clear-cache.
--verbose prints several lines per job, so keep the pipe on the default
URL lines or on --json.

Cache: ~/.cache/vscan. History: ~/.local/share/vscan/history.
Synonyms: ~/.config/vscan/synonyms
  remote = relocate, wfh, work from home

A line replaces the whole synonym group for that word.

Environment:
  HH_TOKEN            optional hh.ru API token
  SUPERJOB_API_KEY    when set, SuperJob uses its API; otherwise HTML

Public feeds that ask for a credit: Remote OK (remoteok.com),
We Work Remotely (weworkremotely.com), Remotive (remotive.com),
Jobicy (jobicy.com), Arbeitnow (arbeitnow.com). stdout is the job URL.
`
}
