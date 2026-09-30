// Package sources wires the concrete job boards without an import cycle.
package sources

import (
	"fmt"
	"os"
	"strings"

	"vscan/internal/httpx"
	"vscan/internal/provider"
	"vscan/internal/provider/arbeitnow"
	"vscan/internal/provider/djinni"
	"vscan/internal/provider/elixirjobs"
	"vscan/internal/provider/fourday"
	"vscan/internal/provider/geekjob"
	"vscan/internal/provider/getmatch"
	"vscan/internal/provider/golangprojects"
	"vscan/internal/provider/habr"
	"vscan/internal/provider/hh"
	"vscan/internal/provider/himalayas"
	"vscan/internal/provider/hn"
	"vscan/internal/provider/jobicy"
	"vscan/internal/provider/jobspresso"
	"vscan/internal/provider/landing"
	"vscan/internal/provider/larajobs"
	"vscan/internal/provider/muse"
	"vscan/internal/provider/nofluff"
	"vscan/internal/provider/nomads"
	"vscan/internal/provider/pythonjobs"
	"vscan/internal/provider/remoteok"
	"vscan/internal/provider/remotive"
	"vscan/internal/provider/superjob"
	"vscan/internal/provider/trudvsem"
	"vscan/internal/provider/wwr"
)

// Names is the default source order.
var Names = []string{
	"hh", "habr", "superjob", "djinni", "getmatch", "geekjob",
	"remoteok", "wwr", "arbeitnow", "remotive", "jobicy", "himalayas", "nomads",
	"nofluff", "landing", "muse", "fourday", "jobspresso",
	"trudvsem", "hn",
	"python", "elixir", "larajobs", "golangprojects",
}

// Canonical returns the official id for a platform name.
func Canonical(name string) (string, bool) {
	for _, n := range Names {
		if strings.EqualFold(n, name) {
			return n, true
		}
	}
	return "", false
}

// Build returns the named providers. An empty list means all of them.
func Build(client *httpx.Client, names []string) ([]provider.Provider, error) {
	if len(names) == 0 {
		names = Names
	}
	var out []provider.Provider
	for _, name := range names {
		switch name {
		case "hh":
			out = append(out, &hh.Provider{HTTP: client, Token: os.Getenv("HH_TOKEN")})
		case "habr":
			out = append(out, &habr.Provider{HTTP: client})
		case "superjob":
			out = append(out, &superjob.Provider{HTTP: client, Key: os.Getenv("SUPERJOB_API_KEY")})
		case "djinni":
			out = append(out, &djinni.Provider{HTTP: client})
		case "getmatch":
			out = append(out, &getmatch.Provider{HTTP: client})
		case "geekjob":
			out = append(out, &geekjob.Provider{HTTP: client})
		case "remoteok":
			out = append(out, &remoteok.Provider{HTTP: client})
		case "wwr":
			out = append(out, &wwr.Provider{HTTP: client})
		case "arbeitnow":
			out = append(out, &arbeitnow.Provider{HTTP: client})
		case "remotive":
			out = append(out, &remotive.Provider{HTTP: client})
		case "jobicy":
			out = append(out, &jobicy.Provider{HTTP: client})
		case "himalayas":
			out = append(out, &himalayas.Provider{HTTP: client})
		case "nomads":
			out = append(out, &nomads.Provider{HTTP: client})
		case "nofluff":
			out = append(out, &nofluff.Provider{HTTP: client})
		case "landing":
			out = append(out, &landing.Provider{HTTP: client})
		case "muse":
			out = append(out, &muse.Provider{HTTP: client})
		case "fourday":
			out = append(out, &fourday.Provider{HTTP: client})
		case "jobspresso":
			out = append(out, &jobspresso.Provider{HTTP: client})
		case "trudvsem":
			out = append(out, &trudvsem.Provider{HTTP: client})
		case "hn":
			out = append(out, &hn.Provider{HTTP: client})
		case "python":
			out = append(out, &pythonjobs.Provider{HTTP: client})
		case "elixir":
			out = append(out, &elixirjobs.Provider{HTTP: client})
		case "larajobs":
			out = append(out, &larajobs.Provider{HTTP: client})
		case "golangprojects":
			out = append(out, &golangprojects.Provider{HTTP: client})
		default:
			return nil, fmt.Errorf("unknown platform %q (available: %s)", name, strings.Join(Names, ", "))
		}
	}
	return out, nil
}
