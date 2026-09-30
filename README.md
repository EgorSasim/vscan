# vscan

Terminal search for programmer vacancies. One static binary for macOS and Linux. Links are printed to stdout as they are found and work in a pipe. There is no UI and no Telegram client inside the binary.

After `make build` every example below uses `./bin/vscan` from the project directory. After `make install` the same commands use `vscan`.

## Contents

1. [Build](#1-build)
2. [Run a search](#2-run-a-search)
3. [Presets](#3-presets)
4. [Query language](#4-query-language)
5. [Boards](#5-boards)
6. [Verbose output and JSON](#6-verbose-output-and-json)
7. [Cache](#7-cache)
8. [Scanner](#8-scanner)
9. [Receive scan results](#9-receive-scan-results)
10. [Aliases](#10-aliases)
11. [History](#11-history)
12. [Exit codes](#12-exit-codes)
13. [Files and environment](#13-files-and-environment)
14. [Auto-apply](#14-auto-apply)

## 1. Build

Go 1.26 or newer. Cgo is not used. From the project directory:

```bash
go version
make build
./bin/vscan --help
```

`make build` passes `-tags noresponse`. That build tag leaves the auto-apply module out of the binary. The same flag is what you want on a VPS:

```bash
CGO_ENABLED=0 go build -tags noresponse -trimpath -ldflags="-s -w" -o bin/vscan ./cmd/vscan
```

`make build` writes only `bin/vscan` inside the project. It does not copy the binary into `/usr/local`, `/Applications`, or `PATH`. The personal binary with auto-apply is a different target, `make build-response` (`-tags response`), and it writes `bin/vscan-response`.

Check the binary with a short live search:

```bash
./bin/vscan --no-history --max-pages 1 --sources remotive 'Go||Golang'
```

Install the `vscan` command. The default prefix is `/usr/local`, which lands in `/usr/local/bin` and usually needs `sudo`:

```bash
sudo make install
vscan --help
```

Install into a directory you own:

```bash
make install PREFIX="$HOME/.local"
```

That writes `$HOME/.local/bin/vscan`. The directory has to be on `PATH`.

Other targets:

```bash
make test    # go test ./...
make cross   # dist/vscan-darwin-amd64, darwin-arm64, linux-amd64, linux-arm64
make clean   # removes bin/ and dist/
```

`make cross` builds the same static binaries for Intel and Apple silicon Macs, and for 64-bit Linux.

## 2. Run a search

A one-shot search prints current matches and exits. It does not read or write the seen-cache.

```bash
./bin/vscan --help
./bin/vscan Go
./bin/vscan Angular
./bin/vscan 'Senior&|React'
./bin/vscan 'Senior&Angular' | wc -l
echo 'Senior&Angular' | ./bin/vscan
```

Limit the work while you try a query. `--max-pages 1` reads one page per board. `--sources` is a comma list. `--no-history` keeps this trial out of the history file.

```bash
./bin/vscan --no-history --max-pages 1 --sources habr,remotive 'Golang||Go'
./bin/vscan --no-history --max-pages 1 --platform='hh||habr' --profession='Angular||TypeScript'
./bin/vscan --company='TBank||AlfaBank' --profession='Golang||Go' --platform=habr
```

Stdout is one URL per line. Board errors go to stderr, and the other boards continue. A closed pipe (`head`) stops the process with exit code 0:

```bash
./bin/vscan --no-history --max-pages 1 --sources remotive 'Go||Golang' | head -n 1
```

## 3. Presets

A language or framework name is a preset. `./bin/vscan Go` searches for Golang and Go. `./bin/vscan Angular` searches for Angular and AngularJS. The same names expand inside a larger query and inside `--profession`:

```bash
./bin/vscan Go
./bin/vscan angular
./bin/vscan 'C++'
./bin/vscan 'C#'
./bin/vscan 'Senior&|React&|Remote'
./bin/vscan --profession=TypeScript --platform='hh||habr'
./bin/vscan presets
```

`./bin/vscan presets` prints every name and the query it becomes. Letter case does not matter. A quoted name stays one word: `./bin/vscan '"Go"'` searches for Go and does not add Golang. An alias with the same name replaces the preset.

The matcher compares words, so `C++` and `C#` also match the word C, and `.NET` also matches the word net. The extra names (`CPP`, `CSharp`, `DotNet`) cover the spellings that are not punctuation.

### Languages

| Type any of | Search |
| --- | --- |
| `Go`, `Golang` | `Golang\|\|Go` |
| `JavaScript`, `JS` | `JavaScript\|\|JS` |
| `TypeScript`, `TS` | `TypeScript\|\|TS` |
| `Python`, `Py` | `Python` |
| `Java` | `Java` |
| `Kotlin` | `Kotlin` |
| `Swift` | `Swift` |
| `Rust` | `Rust` |
| `PHP` | `PHP` |
| `Ruby` | `Ruby` |
| `C` | `C` |
| `C++`, `CPP`, `CPlusPlus` | `C++\|\|CPP\|\|CPlusPlus` |
| `C#`, `CSharp` | `C#\|\|CSharp` |
| `Scala` | `Scala` |
| `Elixir` | `Elixir` |
| `Clojure` | `Clojure` |
| `Dart` | `Dart` |
| `Haskell` | `Haskell` |
| `Lua` | `Lua` |
| `Perl` | `Perl` |
| `R` | `R` |
| `SQL` | `SQL` |
| `Solidity` | `Solidity` |
| `Erlang` | `Erlang` |
| `F#`, `FSharp` | `F#\|\|FSharp` |
| `Objective-C`, `ObjC` | `Objective-C\|\|ObjC` |
| `Groovy` | `Groovy` |
| `MATLAB` | `MATLAB` |
| `Zig` | `Zig` |
| `HTML` | `HTML` |
| `CSS` | `CSS` |
| `Bash`, `Shell` | `Bash\|\|Shell` |
| `PowerShell` | `PowerShell` |

### Frameworks and platforms

| Type any of | Search |
| --- | --- |
| `Angular`, `AngularJS` | `Angular\|\|AngularJS` |
| `React`, `ReactJS`, `React.js` | `React\|\|ReactJS\|\|React.js` |
| `Vue`, `VueJS`, `Vue.js` | `Vue\|\|VueJS\|\|Vue.js` |
| `Svelte`, `SvelteKit` | `Svelte\|\|SvelteKit` |
| `Next`, `NextJS`, `Next.js` | `Next.js\|\|NextJS` |
| `Nuxt`, `NuxtJS`, `Nuxt.js` | `Nuxt\|\|NuxtJS` |
| `Node`, `NodeJS`, `Node.js` | `Node\|\|NodeJS\|\|Node.js` |
| `Express`, `ExpressJS` | `Express\|\|ExpressJS` |
| `Nest`, `NestJS`, `Nest.js` | `NestJS\|\|Nest.js` |
| `Django` | `Django` |
| `Flask` | `Flask` |
| `FastAPI` | `FastAPI` |
| `Spring`, `SpringBoot` | `Spring\|\|SpringBoot` |
| `Rails`, `RubyOnRails`, `RoR` | `Rails\|\|RubyOnRails\|\|RoR` |
| `Laravel` | `Laravel` |
| `Symfony` | `Symfony` |
| `Flutter` | `Flutter` |
| `ReactNative`, `React-Native` | `ReactNative\|\|React-Native` |
| `DotNet`, `.NET` | `DotNet\|\|.NET` |
| `ASP.NET`, `ASPNet` | `ASP.NET` |
| `Blazor` | `Blazor` |
| `MAUI` | `MAUI` |
| `WPF` | `WPF` |
| `WinForms` | `WinForms` |
| `Xamarin` | `Xamarin` |
| `Gin` | `Gin` |
| `Fiber` | `Fiber` |
| `Ktor` | `Ktor` |
| `Phoenix` | `Phoenix` |
| `Actix` | `Actix` |
| `Axum` | `Axum` |
| `Rocket` | `Rocket` |
| `Micronaut` | `Micronaut` |
| `Quarkus` | `Quarkus` |
| `Hibernate` | `Hibernate` |
| `JavaFX` | `JavaFX` |
| `Jetpack`, `Compose` | `Jetpack\|\|Compose` |
| `SwiftUI` | `SwiftUI` |
| `UIKit` | `UIKit` |
| `Electron` | `Electron` |
| `jQuery` | `jQuery` |
| `Redux` | `Redux` |
| `GraphQL` | `GraphQL` |
| `Tailwind` | `Tailwind` |
| `Bootstrap` | `Bootstrap` |
| `Ember` | `Ember` |
| `Remix` | `Remix` |
| `Astro` | `Astro` |
| `SolidJS` | `SolidJS` |
| `Qwik` | `Qwik` |
| `HTMX` | `HTMX` |
| `TensorFlow` | `TensorFlow` |
| `PyTorch` | `PyTorch` |
| `Android` | `Android` |
| `iOS` | `iOS` |
| `Unity` | `Unity` |
| `Unreal` | `Unreal` |

## 4. Query language

The query is one argument. Quote it. Operators may have spaces around them.

| Form | Meaning |
| --- | --- |
| `Senior&Angular` | both words, whole word, no synonyms. `Angular` does not match `AngularJS` |
| `Senior\|\|Angular` | either word is enough |
| `Senior&\|Angular&\|Remote` | smart AND. Words are matched in the title, description, skills, and tags, in any order. `Angular` matches `AngularJS`. `Remote` matches `relocate` and `wfh` |
| `Senior&Angular&\|Remote` | `Senior` and `Angular` are literal, `Remote` uses synonyms |
| `(Senior\|\|Lead)&Angular` | parentheses. `&` and `&\|` bind tighter than `\|\|` |
| `"remote work"` | a phrase |

The operator to the left of a word sets that word's mode. The first word takes the mode of the first operator. One bare word is a literal search after presets expand, so `Go` becomes `Golang||Go` and `"Go"` stays the word Go. Several bare words are an error. A single `|` is an error; OR is `||`.

Field flags use the same operators and are combined with AND. An alias name expands in place of a word.

- `--company` matches the company name only. Hyphens are ignored, so `TBank` matches `T-Bank`.
- `--profession` matches the title and skills, not the full description.
- `--platform` picks boards. A vacancy has one source, so list several with `||`.

Custom synonyms live in `~/.config/vscan/synonyms` (`$XDG_CONFIG_HOME/vscan/synonyms`):

```
remote = relocate, wfh, work from home
```

A line replaces the whole group for that word. The built-in groups also include common translations of remote, senior, middle, and junior.

## 5. Boards

The names are fixed. Any other `--platform` value is an error.

| Name | Board |
| --- | --- |
| `hh` | HeadHunter, `api.hh.ru` |
| `habr` | Habr Career, RSS |
| `superjob` | SuperJob |
| `djinni` | Djinni |
| `getmatch` | getmatch |
| `geekjob` | Geekjob |
| `remoteok` | Remote OK |
| `wwr` | We Work Remotely, programming feed |
| `arbeitnow` | Arbeitnow job-board API |
| `remotive` | Remotive remote jobs |
| `jobicy` | Jobicy remote jobs |
| `himalayas` | Himalayas jobs API |
| `nomads` | Working Nomads |

```bash
./bin/vscan --platform='hh||habr||remotive' --profession='Angular||TypeScript'
./bin/vscan --sources hh,habr 'Senior&Angular'
```

When both `--platform` and `--sources` are set, the result is their intersection. vscan does not bypass a login or a captcha. An empty or blocked page is an error on stderr.

`HH_TOKEN` is an optional hh.ru API token. `SUPERJOB_API_KEY`, when set, makes SuperJob use its API. Otherwise SuperJob is read from public HTML.

These feeds ask for a credit. stdout is the job URL. The feeds: [remoteok.com](https://remoteok.com), [weworkremotely.com](https://weworkremotely.com), [remotive.com](https://remotive.com), [jobicy.com](https://jobicy.com), [arbeitnow.com](https://www.arbeitnow.com).

## 6. Verbose output and JSON

The default line is a URL, so a pipe stays one job per line.

```bash
./bin/vscan --verbose --no-history --max-pages 1 --sources remotive 'Go||Golang'
./bin/vscan --json --no-history --max-pages 1 --sources jobicy 'Go||Golang'
```

`--verbose` (`-v`) prints a block. Known fields are title, company, source, remote, location, salary, posted time, and age in days. Unknown fields are left out. `remote: no` is printed only when the board says the job is not remote.

```
https://remotive.com/remote-jobs/software-development/example
title: Senior Go Developer
company: Example
source: remotive
remote: yes
location: Europe
posted: 2026-09-18T12:23:56Z
age: 11 days
```

`--json` prints one object per line with `url`, `title`, `company`, `source`, and the same extra fields when they are known (`remote`, `location`, `salary`, `posted`, `age`). `--json` together with `--verbose` stays JSON.

## 7. Cache

The seen-cache remembers vacancy URLs so a scanner does not print the same link on every pass.

| | |
| --- | --- |
| Path | `~/.cache/vscan/seen/` or `$XDG_CACHE_HOME/vscan/seen/` |
| What is stored | one URL per line, in a file named by the search and the board list |
| Who uses it | `--scanner` / `--scan` only |
| One-shot search | `./bin/vscan 'Go'` always prints the current matches and ignores this cache |

On a scanner start:

1. The seen file for this search is empty. The first pass remembers every current match and prints nothing. stderr, when it is a terminal, says the baseline was saved.
2. Later passes print a URL only when it was not in the file.
3. You stop the scanner and start the same command again while the file still has URLs. The new process prints links that are not in the file on the first pass. It does not take another silent pass.
4. You change the query, the company, the profession, the platform, or the source list. That is a different file, so the new search starts with an empty baseline.

Clear every remembered URL. History and aliases stay.

```bash
./bin/vscan --clear-cache
```

Stdout is one line, for example `cleared 12 remembered links`, and the process exits 0. The count is the number of stored links, across every saved search.

Clear and immediately show the current matches, then keep watching:

```bash
./bin/vscan --clear-cache --scanner --every 60 \
  --platform=remoteok 'Go||Golang'
```

`--clear-cache` together with `--scanner` deletes the cache, prints the matches that exist now, stores them again, and after that prints only new URLs. Run those two flags in one command. A later `./bin/vscan --scanner ...` with an empty cache goes back to a silent first pass.

`--clear-cache` removes the whole `seen` directory. It forgets every scanner query, not only the one on the command line.

## 8. Scanner

`--scan` is the same flag as `--scanner`. The process repeats the search until Ctrl+C. `--every N` is the gap in minutes. The default is 60. `--every`, `--listen`, and `--webhook` require `--scanner`.

```bash
./bin/vscan --scanner --every 30 'Senior&|Angular'
```

For a short check, use one board, one page, and a one-minute gap. Leave this running:

```bash
./bin/vscan --clear-cache --scanner --every 1 --no-history --max-pages 1 \
  --platform=remoteok 'Go||Golang'
```

Ctrl+C stops it with exit code 0. Day to day, use `--every 30` or `--every 60` and drop `--clear-cache` after the first run, so only new vacancies are reported.

Stdout still receives every emitted URL. The sections below add Telegram, SSE, or a webhook on top of that. All three can run in the same command. A scanner pass also writes progress to stderr when stderr is a terminal.

## 9. Receive scan results

Telegram is not inside vscan. The three outputs are the stdout pipe, the SSE port (`--listen`), and an HTTP POST (`--webhook`).

### 9.1 Telegram bot

Create a bot with [@BotFather](https://t.me/BotFather) and copy the token. Open the bot in Telegram and send it any message, then read your chat id:

```bash
export TOKEN='123456:ABC-your-token'
curl -s "https://api.telegram.org/bot$TOKEN/getUpdates"
```

Use the number in `chat.id`.

```bash
export TOKEN='123456:ABC-your-token'
export CHAT='123456789'
./bin/vscan --clear-cache --scanner --every 1 --no-history --max-pages 1 \
  --platform=remoteok 'Go||Golang' | while read -r url; do
  curl -s -X POST "https://api.telegram.org/bot$TOKEN/sendMessage" \
    -d chat_id="$CHAT" --data-urlencode text="$url"
done
```

Each new URL becomes one Telegram message. `--clear-cache` makes the first pass send the vacancies that already exist. Without it, the first pass is silent and the bot gets a message only when a later pass finds a new URL.

Keep the default one-URL-per-line stdout. `--verbose` prints several lines per job and breaks `while read`. For a richer message, use `--json` and `jq`:

```bash
./bin/vscan --clear-cache --json --scanner --every 1 --no-history --max-pages 1 \
  --platform=remoteok 'Go||Golang' | while read -r line; do
  text=$(printf '%s' "$line" | jq -r '"\(.title) — \(.company)\n\(.url)"')
  curl -s -X POST "https://api.telegram.org/bot$TOKEN/sendMessage" \
    -d chat_id="$CHAT" --data-urlencode text="$text"
done
```

`--every 1` is for this check. For a real watch, use 30 or 60.

### 9.2 SSE (`--listen`)

`--listen` opens a local HTTP server. The default address is `127.0.0.1:8787`. Only loopback is accepted (`127.0.0.1` or `localhost`). `0.0.0.0` is rejected.

Terminal 1, leave it running:

```bash
./bin/vscan --clear-cache --scanner --every 1 --no-history --max-pages 1 \
  --listen 127.0.0.1:8787 --platform=remoteok 'Go||Golang'
```

stderr prints `vscan: events at http://127.0.0.1:8787/events`.

Terminal 2:

```bash
curl -s http://127.0.0.1:8787/health
curl -N http://127.0.0.1:8787/events
```

`/health` returns `{"status":"ok"}`. `/events` is a Server-Sent Events stream. Each vacancy is one event:

```
event: vacancy
data: {"url":"https://remoteok.com/remote-jobs/1","title":"Go Developer","company":"Acme","source":"remoteok","remote":"yes"}
```

Connect `curl -N` before or during a pass. Events that fire while nobody is connected are not replayed. stdout still prints the same URLs in terminal 1.

### 9.3 Webhook (`--webhook`)

`--webhook` POSTs one JSON object per new vacancy to a URL you run. The body is the same object as `--json`, with `Content-Type: application/json`. A failed POST is retried twice. Your handler should answer with a 2xx status.

Terminal 1, a local receiver on port 8790:

```bash
python3 - <<'PY'
from http.server import BaseHTTPRequestHandler, HTTPServer

class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get("Content-Length", 0))
        print(self.rfile.read(n).decode(), flush=True)
        self.send_response(204)
        self.end_headers()

    def log_message(self, fmt, *args):
        return

HTTPServer(("127.0.0.1", 8790), Handler).serve_forever()
PY
```

Terminal 2:

```bash
./bin/vscan --clear-cache --scanner --every 1 --no-history --max-pages 1 \
  --webhook http://127.0.0.1:8790/vacancy \
  --platform=remoteok 'Go||Golang'
```

Terminal 1 prints one JSON object per vacancy. Point `--webhook` at your own service the same way. The URL has to be reachable from the machine running vscan.

### 9.4 All three together

stdout, SSE, and the webhook can run in one process:

```bash
./bin/vscan --clear-cache --scanner --every 30 \
  --listen 127.0.0.1:8787 \
  --webhook http://127.0.0.1:8790/vacancy \
  'Senior&|Angular'
```

The pipe in section 9.1 can wrap this command as well. A closed stdout pipe stops the whole scanner.

## 10. Aliases

A short name expands to an expression in the query and in flags. A quoted phrase is not expanded.

```bash
./bin/vscan alias myStack='Angular||Typescript||JavaScript||JS||TS||React'
./bin/vscan --profession=myStack --platform='hh||habr'
./bin/vscan alias
./bin/vscan alias --delete myStack
```

File `~/.config/vscan/aliases`, one line `myStack=Angular||Typescript||JavaScript||JS||TS||React`. A name starts with a letter or `_`, then letters, digits, and `_`. A cycle is an error. An alias whose name matches a preset replaces that preset.

## 11. History

```bash
./bin/vscan --history
./bin/vscan --history 3
./bin/vscan --no-history 'Senior&Angular'
```

File `~/.local/share/vscan/history` (`$XDG_DATA_HOME/vscan/history`). New queries go to the top. A repeat moves to the top and is not stored twice. The saved line is the query before alias expansion, so aliases expand again on `--history N`. `--history` without a number lists queries and does not search. `--clear-cache` does not delete this file.

## 12. Exit codes

| Code | When |
| --- | --- |
| 0 | at least one match, the scanner was stopped, `--clear-cache` finished, or the pipe was closed (`head`) |
| 1 | a one-shot search found nothing |
| 2 | bad arguments, an unknown board, or a broken query |

Run the built binary when you check these codes. `go run` wraps the real status.

## 13. Files and environment

| Path | Role |
| --- | --- |
| `~/.cache/vscan/seen/` | scanner memory of vacancy URLs |
| `~/.local/share/vscan/history` | past queries |
| `~/.config/vscan/aliases` | alias definitions |
| `~/.config/vscan/synonyms` | synonym groups |

`$XDG_CACHE_HOME`, `$XDG_DATA_HOME`, and `$XDG_CONFIG_HOME` replace the three home directories above.

| Variable | Role |
| --- | --- |
| `HH_TOKEN` | optional hh.ru API token |
| `SUPERJOB_API_KEY` | SuperJob API. Without it, SuperJob is read from HTML |

## 14. Auto-apply

Auto-apply lives in `internal/response`. The VPS build passes `-tags noresponse`, so that package is not linked and `creds` / `--response` cannot store a login:

```bash
make build
# or
CGO_ENABLED=0 go build -tags noresponse -o bin/vscan ./cmd/vscan
```

`make cross` uses the same tag. If both `-tags response` and `-tags noresponse` are set, `noresponse` wins.

A personal Mac or Linux machine builds a second binary. `./bin/vscan` stays the search program, so copying it to a VPS cannot store logins.

```bash
make build-response
./bin/vscan-response creds set hh
```

`creds set` asks for the login and the password on the terminal. The password is not printed and is not written into the project. macOS stores it in the Keychain (service `vscan/hh`). Linux stores it in Secret Service via `secret-tool` (package `libsecret-tools`). On macOS the password is handed to the `security` tool as an argument, so it can appear briefly in the process list; it is not saved in a file. `creds` prints board names only. `creds delete hh` removes that login.

Boards that have their own apply button: `hh`, `habr`, `superjob`, `djinni`, `getmatch`, `geekjob`. Aggregators such as Remote OK and Remotive are skipped, because their links leave the board. Chrome or Chromium must be installed. The browser logs in, then clicks Apply. The resume already chosen on the site is the one that is sent. There is no cover letter in this phase.

A captcha, a confirmation code, or a form with extra questions is skipped. The reason goes to stderr and the next URL is tried.

Pass links one per line, or as one argument split by the unit separator (a character that cannot appear in a URL):

```bash
./bin/vscan --no-history --max-pages 1 --sources hh,habr Go > /tmp/jobs.txt
./bin/vscan-response --response < /tmp/jobs.txt

printf '%s\x1f%s\n' \
  'https://hh.ru/vacancy/1' \
  'https://career.habr.com/vacancies/2' | ./bin/vscan-response --response

./bin/vscan-response --headed --response "$(printf '%s\x1f%s' \
  'https://hh.ru/vacancy/1' \
  'https://djinni.co/jobs/2')"
```

`--headed` shows the browser window. Stdout of a finished run is `applied N`. Exit 0 means at least one application was confirmed, 1 means none, 2 means bad arguments or a build without this module.

