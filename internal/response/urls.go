package response

import (
	"fmt"
	"net/url"
	"strings"
)

// URLSeparator splits several vacancy links in one argument.
// It cannot appear in a URL. Newlines are also separators.
const URLSeparator = "\x1f"

// ApplyBoards are the boards that have their own apply button.
var ApplyBoards = []string{"hh", "habr", "superjob", "djinni", "getmatch", "geekjob"}

// Link is one vacancy URL from a response list.
type Link struct {
	Raw     string
	URL     *url.URL
	Board   string
	Apply   bool
	Problem string
}

// ParseLinks splits text on newlines and URLSeparator.
func ParseLinks(text string) []Link {
	text = strings.ReplaceAll(text, URLSeparator, "\n")
	var out []Link
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, classify(line))
	}
	return out
}

func classify(raw string) Link {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return Link{Raw: raw, Problem: "not an http(s) URL"}
	}
	board, apply, ok := boardOf(u.Hostname())
	if !ok {
		return Link{Raw: raw, URL: u, Problem: "unknown board"}
	}
	if !apply {
		return Link{Raw: raw, URL: u, Board: board, Problem: "this board has no apply button"}
	}
	return Link{Raw: raw, URL: u, Board: board, Apply: true}
}

func boardOf(host string) (id string, apply bool, ok bool) {
	host = strings.ToLower(host)
	host = strings.TrimPrefix(host, "www.")
	switch {
	case host == "hh.ru" || strings.HasSuffix(host, ".hh.ru") ||
		host == "hh.kz" || strings.HasSuffix(host, ".hh.kz") ||
		host == "hh.by" || strings.HasSuffix(host, ".hh.by") ||
		host == "hh.uz" || strings.HasSuffix(host, ".hh.uz"):
		return "hh", true, true
	case host == "career.habr.com":
		return "habr", true, true
	case host == "superjob.ru" || strings.HasSuffix(host, ".superjob.ru"):
		return "superjob", true, true
	case host == "djinni.co" || strings.HasSuffix(host, ".djinni.co"):
		return "djinni", true, true
	case host == "getmatch.ru" || strings.HasSuffix(host, ".getmatch.ru"):
		return "getmatch", true, true
	case host == "geekjob.ru" || strings.HasSuffix(host, ".geekjob.ru"):
		return "geekjob", true, true
	case host == "remoteok.com" || strings.HasSuffix(host, ".remoteok.com"):
		return "remoteok", false, true
	case host == "weworkremotely.com" || strings.HasSuffix(host, ".weworkremotely.com"):
		return "wwr", false, true
	case host == "remotive.com" || strings.HasSuffix(host, ".remotive.com"):
		return "remotive", false, true
	case host == "jobicy.com" || strings.HasSuffix(host, ".jobicy.com"):
		return "jobicy", false, true
	case host == "himalayas.app" || strings.HasSuffix(host, ".himalayas.app"):
		return "himalayas", false, true
	case host == "arbeitnow.com" || strings.HasSuffix(host, ".arbeitnow.com"):
		return "arbeitnow", false, true
	case host == "workingnomads.com" || strings.HasSuffix(host, ".workingnomads.com"):
		return "nomads", false, true
	case host == "nofluffjobs.com" || strings.HasSuffix(host, ".nofluffjobs.com"):
		return "nofluff", false, true
	case host == "landing.jobs" || strings.HasSuffix(host, ".landing.jobs"):
		return "landing", false, true
	case host == "themuse.com" || strings.HasSuffix(host, ".themuse.com"):
		return "muse", false, true
	case host == "4dayweek.io" || strings.HasSuffix(host, ".4dayweek.io"):
		return "fourday", false, true
	case host == "jobspresso.co" || strings.HasSuffix(host, ".jobspresso.co"):
		return "jobspresso", false, true
	case host == "trudvsem.ru" || strings.HasSuffix(host, ".trudvsem.ru"):
		return "trudvsem", false, true
	case host == "news.ycombinator.com":
		return "hn", false, true
	case host == "python.org" || strings.HasSuffix(host, ".python.org"):
		return "python", false, true
	case host == "elixirjobs.net" || strings.HasSuffix(host, ".elixirjobs.net"):
		return "elixir", false, true
	case host == "larajobs.com" || strings.HasSuffix(host, ".larajobs.com"):
		return "larajobs", false, true
	case host == "golangprojects.com" || strings.HasSuffix(host, ".golangprojects.com"):
		return "golangprojects", false, true
	default:
		return "", false, false
	}
}

// KnownBoard reports whether platform is one that can store a login.
func KnownBoard(platform string) bool {
	for _, id := range ApplyBoards {
		if id == platform {
			return true
		}
	}
	return false
}

// BoardList is the help list of apply boards.
func BoardList() string {
	return strings.Join(ApplyBoards, ", ")
}

func unknownBoard(platform string) error {
	return fmt.Errorf("unknown apply board %q (%s)", platform, BoardList())
}
