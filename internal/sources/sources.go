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
	"vscan/internal/provider/geekjob"
	"vscan/internal/provider/getmatch"
	"vscan/internal/provider/habr"
	"vscan/internal/provider/hh"
	"vscan/internal/provider/himalayas"
	"vscan/internal/provider/jobicy"
	"vscan/internal/provider/nomads"
	"vscan/internal/provider/remoteok"
	"vscan/internal/provider/remotive"
	"vscan/internal/provider/superjob"
	"vscan/internal/provider/wwr"
)

// Names is the default source order.
var Names = []string{
	"hh", "habr", "superjob", "djinni", "getmatch", "geekjob",
	"remoteok", "wwr", "arbeitnow", "remotive", "jobicy", "himalayas", "nomads",
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
		default:
			return nil, fmt.Errorf("unknown platform %q (available: %s)", name, strings.Join(Names, ", "))
		}
	}
	return out, nil
}
