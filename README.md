# vscan

Terminal search for programmer vacancies. One static binary for macOS and Linux. Links are printed to stdout as they are found and work in a pipe. There is no UI and no Telegram client inside the binary.

After `make build` every example below uses `./bin/vscan` from the project directory. After `make install` the same commands use `vscan`.

## Contents

1. [Build](#1-build)
2. [Usage](#2-usage)
3. [Presets](#3-presets)
4. [Query language](#4-query-language)
5. [Boards](#5-boards)
6. [Verbose output and JSON](#6-verbose-output-and-json)
7. [Cache](#7-cache)
8. [Scanner](#8-scanner)
9. [Telegram bot](#9-telegram-bot)
10. [Receive scan results](#10-receive-scan-results)
11. [Aliases](#11-aliases)
12. [History](#12-history)
13. [Exit codes](#13-exit-codes)
14. [Files and environment](#14-files-and-environment)
15. [Auto-apply](#15-auto-apply)

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
./bin/vscan --no-history --max-pages 1 --sources remotive 'Go|Golang'
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

## 2. Usage

This section is the command reference: synopsis, the query operand, every flag and the values it accepts, subcommands, streams, signals, and the errors you will actually see. Later sections show the same features as longer recipes.

A one-shot search prints current matches and exits. It does not read or write the seen-cache. A scanner (`--scanner`) repeats until you stop it.

### Synopsis

```
vscan [flags] [query]
vscan presets
vscan alias [NAME=EXPRESSION]
vscan alias --delete NAME
vscan creds [set|delete PLATFORM]
vscan --response [URLS]
```

`vscan` below means `./bin/vscan` before install and `vscan` after `make install`. The query is one argument. Quote it when it contains `&`, `|`, spaces, or parentheses.

```bash
./bin/vscan --help
./bin/vscan Go
./bin/vscan Angular
./bin/vscan 'Senior&|React'
./bin/vscan 'Senior&Angular' | wc -l
echo 'Senior&Angular' | ./bin/vscan
./bin/vscan --no-history --max-pages 1 --sources habr,remotive 'Golang|Go'
./bin/vscan --no-history --max-pages 1 --platform='hh|habr' --profession='Angular|TypeScript'
./bin/vscan --company='TBank|AlfaBank' --profession='Golang|Go' --platform=habr
./bin/vscan --no-history --max-pages 1 --sources remotive 'Go|Golang' | head -n 1
```

### How flags are written

A flag is a word that starts with `-`. Anything else is the query, unless it is the subcommand `alias`, `presets`, or `creds` in the first position.

| Form | Meaning |
| --- | --- |
| `--json`, `-v`, `--scanner` | a switch. It takes no value. `--json=true` is an unknown flag |
| `--max-pages 1` and `--max-pages=1` | the same option. The next word is the value, unless that word starts with `-` |
| `--listen` | the value is optional. Alone, the address is `127.0.0.1:8787` |
| `--history` | the number is optional. The next word is read as the number only when it is an integer |
| `--response` | the URL list is optional. Omit it to read stdin |
| `--` | end of flags. The next word is the query even if it starts with `-` |
| `-h`, `--help` | print help and exit 0. Later arguments are not read |

```bash
./bin/vscan --max-pages=1 --sources=remotive Go
./bin/vscan --max-pages 1 --sources remotive Go
./bin/vscan -- --help
```

The last command searches for the word `--help`. It does not print the help text.

Short options are only `-h` and `-v`. There is no `-j`, no combined `-vh`, and no `--version`.

### Operand: the query

One positional argument. Several words without quotes are an error (`the query must be one argument; quote it`).

| Input | What runs |
| --- | --- |
| `Go` | preset, becomes `Golang\|Go` |
| `angular` | same preset as `Angular`. Case does not matter |
| `'Senior&Angular'` | both words, whole word |
| `'Senior\|Angular'` | either word |
| `'Senior&\|Angular&\|Remote'` | smart AND. See [Query language](#4-query-language) |
| `'Angular&!React'` | Angular, and not the word React |
| `'"Go"'` | the word Go only. The preset is not applied |
| no argument, and a pipe | the first line of stdin is the query |
| no argument, and a terminal | error, exit 2. Pass a query or `--company`, `--profession`, or `--platform` |
| `--company=Acme` and no query | legal. The company filter is the whole search |

A field flag and a query are combined with AND. Every one that is set must match.

```bash
./bin/vscan 'Senior&Angular&Typescript'
./bin/vscan '(Senior|Lead)&Angular'
./bin/vscan '"remote work"'
./bin/vscan 'Angular&!"full stack"&!fullstack&!.NET'
echo 'Senior&|Angular' | ./bin/vscan --no-history --max-pages 1 --sources remotive
```

`&&` is two `&` operators with nothing between them. The error is `operator needs a word or a parenthesis after it`. AND is one `&`. OR is one `|`. `||` is two OR operators, and the error is `| needs a word or a parenthesis`. `!` drops the next word, phrase, or group: `Angular&!React`. A bare `!` fails with the same missing-word error. Quote a query that contains `|` or `!`. Without quotes the shell pipes on `|`, and in double quotes `!` is history expansion. Single quotes avoid both.

### Options

#### `-h`, `--help`

Print the built-in help to stdout and exit 0. No search runs.

```bash
./bin/vscan -h
./bin/vscan --help
```

#### `--json`

One JSON object per line instead of a bare URL. Fields that the board did not provide are omitted.

| Field | When it is present |
| --- | --- |
| `url`, `title`, `company`, `source` | always |
| `remote` | `"yes"` or `"no"` when the board says so, or `"yes"` when the text says remote |
| `location`, `salary` | when the board published them |
| `posted` | RFC3339 time |
| `age` | `today`, `1 day`, or `N days` |

`--json` together with `--verbose` stays JSON. There is no pretty-printed form.

```bash
./bin/vscan --json --no-history --max-pages 1 --sources jobicy 'Go|Golang'
./bin/vscan --json --verbose --no-history --max-pages 1 --sources remotive Go
```

#### `--verbose`, `-v`

A text block per vacancy, then a blank line. The same fields as `--json`. Unknown fields are left out. `remote: no` is printed only when the board says the job is not remote. Absence is not printed as `no`.

Do not pipe `--verbose` into `while read`. One vacancy is several lines. Use the default URL lines, or `--json`.

```bash
./bin/vscan -v --no-history --max-pages 1 --sources remotive Go
./bin/vscan --verbose --no-history --max-pages 1 --sources habr 'Go|Golang'
```

#### `--company EXPR`

Match the company name only. `EXPR` is the query language. Hyphens are ignored, so `TBank` matches `T-Bank`. Presets are not expanded here. `Go` stays the word Go.

```bash
./bin/vscan --company=Acme Go
./bin/vscan --company='TBank|AlfaBank' --profession='Golang|Go'
./bin/vscan --company='T-Bank' 'Senior&|Go'
```

#### `--profession EXPR`

Match the title and the skills. The full description is not searched by this flag. Presets expand, so `--profession=TypeScript` becomes `TypeScript|TS`. Combined with the query and `--company` by AND.

```bash
./bin/vscan --profession=Angular
./bin/vscan --profession='Angular|TypeScript' --platform='hh|habr|nofluff'
./bin/vscan --profession=Go --company=Acme
```

#### `--platform EXPR`

Pick boards. A vacancy has one source, so list several with `|`. `&` on two board names almost never matches.

Names, in any case: `hh`, `habr`, `superjob`, `djinni`, `getmatch`, `geekjob`, `remoteok`, `wwr`, `arbeitnow`, `remotive`, `jobicy`, `himalayas`, `nomads`, `nofluff`, `landing`, `muse`, `fourday`, `jobspresso`, `trudvsem`, `hn`, `python`, `elixir`, `larajobs`, `golangprojects`.

An unknown name is exit 2 and the error lists the known names. Presets are not expanded. The board table is in [Boards](#5-boards).

When `--platform` and `--sources` are both set, the search uses the intersection. An empty intersection is `no platform selected`.

```bash
./bin/vscan --platform=hh 'Senior&Angular'
./bin/vscan --platform='hh|habr|remotive|nofluff' --profession=Angular
./bin/vscan --platform=Remotive --sources=remotive,jobicy Go
./bin/vscan --platform='remoteok|wwr' --sources=wwr Go
```

The third command searches Remotive only. The fourth searches We Work Remotely only.

#### `--sources LIST`

The same board names as `--platform`, comma-separated. Spaces around commas are ignored. Operators are not allowed: `hh|habr` is one illegal name. An empty list is an error.

Default, when the flag is omitted: every board in the table.

```bash
./bin/vscan --sources hh,habr 'Senior&Angular'
./bin/vscan --sources=nofluff,landing,muse,fourday,jobspresso 'Angular|TypeScript'
./bin/vscan --sources remoteok Go
```

#### `--where LIST`

Where to search. Comma-separated. Spaces around commas are ignored. Default, when the flag is omitted: `sites,telegram`.

| Value | Meaning |
| --- | --- |
| `sites` | the job boards in [Boards](#5-boards) |
| `telegram` | public Telegram channel previews |

```bash
./bin/vscan --where sites 'Angular&TypeScript'
./bin/vscan --where=telegram 'Angular&TypeScript'
./bin/vscan --where sites,telegram 'Angular&TypeScript'
```

`--sources` and `--platform` apply to site boards. They are an error when `--where` does not include `sites`. A `--platform` filter does not hide Telegram posts.

The Telegram reader is `internal/provider/telegram`. It reads these public previews: `it_remote`, `remote_jobs_ru`, `itjobsfeed`, `devvacancy`, `remoteit`, `remoteok`, `java_vacancies`, `python_vacancies`, `qa_vacancies`, `devops_vacancies`. With `--max-pages 0` each channel stops after three preview pages. Delete that package and the telegram block in `cmd/vscan/main.go` to remove channel search.

#### `--max-age N`

Drop a vacancy published more than `N` days ago. `N` is a whole number, at least 1. The flag is off unless it is set. A vacancy with no publication date is kept, because its age is unknown.

```bash
./bin/vscan --max-age 5 'Angular&TypeScript'
./bin/vscan --max-age=2 --where sites Go
```

#### `--max-pages N`

How many pages to read from each board. `N` is a whole number, `0` or greater. Default `0`.

| Value | Meaning |
| --- | --- |
| `0` | read until that board reports the end, then apply the safety cap below |
| `1`, `2`, ... | stop after that many pages, even if the board has more |

Safety caps when `N` is `0`, so a feed that never ends cannot run forever:

| Boards | Cap |
| --- | --- |
| `hh`, SuperJob API | until the API reports the last page |
| SuperJob HTML | 20 pages |
| `djinni` | 10 pages |
| `geekjob` | 30 pages |
| `arbeitnow`, `himalayas`, `nofluff`, `landing`, `muse`, `fourday`, `jobspresso`, `trudvsem` | 5 pages |
| `habr`, `remoteok`, `wwr`, `remotive`, `jobicy`, `nomads`, `hn`, `python`, `elixir`, `larajobs`, `golangprojects` | one payload. The flag does not add pages |

```bash
./bin/vscan --max-pages 1 --sources remotive Go
./bin/vscan --max-pages=3 --sources nofluff,landing 'Angular&TypeScript'
./bin/vscan --max-pages 0 --sources hh Go
```

#### `--scanner`, `--scan`

Repeat the search until Ctrl+C. `--scan` is the same flag. The first pass is silent when the seen-file for this search is empty: it only remembers the current URLs. Later passes print a URL only when it is new. See [Scanner](#8-scanner) and [Cache](#7-cache).

`--every`, `--listen`, and `--webhook` require this flag.

```bash
./bin/vscan --scanner --every 30 'Senior&|Angular'
./bin/vscan --scan --every 60 --platform=remoteok Go
```

#### `--every N`

Minutes between scanner passes. `N` is a whole number, at least `1`. Default `60` when `--scanner` is set and `--every` is omitted. Without `--scanner` the flag is an error.

```bash
./bin/vscan --scanner --every 1 --max-pages 1 --sources remoteok Go
./bin/vscan --scanner --every=15 --no-history 'Senior&|React'
```

`--every 1` is for a short check. Day to day, use `30` or `60`.

#### `--listen [ADDR]`

With `--scanner`, open a local HTTP server and publish each new vacancy as Server-Sent Events. Requires `--scanner`.

| Form | Address |
| --- | --- |
| `--listen` | `127.0.0.1:8787` |
| `--listen 127.0.0.1:8787` | that address |
| `--listen=localhost:8787` | that address |
| `--listen 8787` | rejected. A bare port is not `host:port` |

The host must be loopback: `127.0.0.1` or `localhost`. `0.0.0.0` is rejected.

| Method and path | Body |
| --- | --- |
| `GET /health` | `{"status":"ok"}` |
| `GET /events` | SSE. Each vacancy is `event: vacancy` and one JSON `data:` line |

Events that fire while nobody is connected are not replayed. stdout still prints the same URLs. The recipe is in [SSE](#101-sse---listen).

```bash
./bin/vscan --scanner --every 30 --listen 'Senior&|Angular'
./bin/vscan --scanner --listen 127.0.0.1:8787 --max-pages 1 --sources remoteok Go
curl -s http://127.0.0.1:8787/health
curl -N http://127.0.0.1:8787/events
```

#### `--webhook URL`

With `--scanner`, POST one JSON object per new vacancy. `Content-Type` is `application/json`. The body is the same object as `--json`. A failed POST is retried twice. The handler should answer `2xx`. Requires `--scanner`. Can be combined with `--listen` and stdout.

```bash
./bin/vscan --scanner --every 30 --webhook http://127.0.0.1:8790/vacancy Go
./bin/vscan --scanner --listen --webhook http://127.0.0.1:8790/vacancy 'Senior&|Angular'
```

#### `--history`, `--history N`

| Form | Effect |
| --- | --- |
| `--history` | print saved queries, newest first, as `N<TAB>query`. Does not search. Exit 0. An empty file prints `vscan: history is empty` on stderr and exits 0 |
| `--history 3` or `--history=3` | run entry 3 again. Numbers start at 1. Do not also pass a query or a field flag |
| a missing number | exit 2 |

The saved line is the text before alias and preset expansion, so both expand again on replay. `--no-history` on the replayed command is not part of the saved line. The file is described in [History](#12-history).

```bash
./bin/vscan --history
./bin/vscan --history 1
./bin/vscan --history=2 --max-pages 1 --sources remotive
```

The third command is legal: `--max-pages` and `--sources` are not part of the saved search, so they apply to the replay.

#### `--no-history`

Do not write this query into the history file. Does not delete existing lines.

```bash
./bin/vscan --no-history --max-pages 1 Go
```

#### `--clear-cache`

Delete every file under `~/.cache/vscan/seen/`. History and aliases stay. The count is every remembered link, for every search, not one query.

| With | Effect |
| --- | --- |
| alone | print `cleared N remembered links` and exit 0 |
| `--scanner` and a query | delete the cache, print the current matches once, then only new URLs |
| a one-shot search | the seen-cache is still deleted, then the search prints current matches. A one-shot search does not use the cache |

```bash
./bin/vscan --clear-cache
./bin/vscan --clear-cache --scanner --every 60 --platform=remoteok Go
```

#### `--response [URLS]`

Apply to vacancy URLs. This does not search. A query, `--scanner`, or `--history` together with `--response` is an error.

Included only in `bin/vscan-response` (`make build-response`, `-tags response`). `./bin/vscan` from `make build` prints `auto-apply is not included in this build` and exits 2.

| Form | URLs come from |
| --- | --- |
| `--response` | stdin, when stdin is not a terminal. One URL per line |
| `--response URLS` or `--response=URLS` | that argument. Split on newlines and on the unit separator `$'\x1f'` |
| a pipe and an argument together | both lists |

Stdout of a finished run is `applied N`. Exit 0 when `N > 0` or the run was interrupted. Exit 1 when nothing was confirmed. Exit 2 on usage errors. Details and examples are in [Auto-apply](#15-auto-apply).

```bash
./bin/vscan-response --response < /tmp/jobs.txt
printf '%s\x1f%s\n' 'https://hh.ru/vacancy/1' 'https://career.habr.com/vacancies/2' | ./bin/vscan-response --response
```

#### `--headed`

Show the browser window. Legal only together with `--response`. Without `--response` it is an error.

```bash
./bin/vscan-response --headed --response 'https://hh.ru/vacancy/1'
```

### Subcommands

The subcommand is the first argument. `./bin/vscan --json presets` searches for the word `presets`.

#### `presets`

Print every preset as `names = expression` and exit 0. Extra arguments are an error. The tables are in [Presets](#3-presets).

```bash
./bin/vscan presets
./bin/vscan presets | head
```

#### `alias`

| Form | Effect |
| --- | --- |
| `alias` | list `name=expression` lines. No aliases: `vscan: no aliases` on stderr, exit 0 |
| `alias NAME=EXPRESSION` | save a definition. Also `alias NAME EXPRESSION` as two arguments |
| `alias --delete NAME` or `alias -d NAME` | remove that name. Exit 2 if it is missing |
| a bad expression | not saved. Exit 2 |

A name matches `^[A-Za-z_][A-Za-z0-9_]*$`. The body is the query language. A cycle is an error. A user alias replaces a preset of the same name. Quoted phrases in a search are not expanded. The file is in [Aliases](#11-aliases).

```bash
./bin/vscan alias myStack='Angular|Typescript|JavaScript|JS|TS|React'
./bin/vscan alias
./bin/vscan --profession=myStack --platform='hh|habr'
./bin/vscan alias --delete myStack
./bin/vscan alias -d myStack
```

#### `creds`

Store a board login. Same build rule as `--response`: the search binary refuses it.

| Form | Effect |
| --- | --- |
| `creds` | list board names that have a saved login. Passwords are not printed |
| `creds set PLATFORM` | ask for the login and the password on the terminal. Echo of the password is off |
| `creds delete PLATFORM` | remove that login |

`PLATFORM` is one of `hh`, `habr`, `superjob`, `djinni`, `getmatch`, `geekjob`. Any other name is exit 2.

```bash
./bin/vscan-response creds set hh
./bin/vscan-response creds
./bin/vscan-response creds delete hh
```

### Standard input

| Invocation | Stdin |
| --- | --- |
| a query was passed | stdin is not read |
| no query and no field flag, stdin is a pipe | the first line is the query. Later lines are ignored |
| `--response` and stdin is a pipe | every non-empty line is a URL |
| `--response` and stdin is a terminal, and no URL argument | exit 2, nothing to apply |
| `creds set` | the login and the password are read from the terminal, not from the pipe |

### Standard output

Line-buffered and flushed, so a pipe sees each line as it is found.

| Mode | One vacancy |
| --- | --- |
| default | one URL |
| `--verbose` | a block, then a blank line |
| `--json` | one JSON object |
| `--history` | `N` and a tab and the saved query |
| `presets` | `names = expression` |
| `alias` | `name=expression` |
| `--clear-cache` alone | `cleared N remembered links` |
| `--response` | `applied N` |

### Standard error

Messages start with `vscan: `. A board that fails (HTTP error, empty anti-bot page, a captcha on apply) is reported here and the other boards or URLs continue. The process does not exit 2 for a single board failure.

Progress lines are written only when stderr is a terminal. A redirect or a pipe on stderr hides them. There is no `--color` and no `--quiet`.

### Signals

| Signal | Effect |
| --- | --- |
| SIGINT (Ctrl+C), SIGTERM | cancel the search or the apply run. Exit 0 |
| SIGPIPE | ignored as a death signal. A closed stdout, as in `head`, stops the process with exit 0 |

```bash
./bin/vscan --no-history --max-pages 1 --sources remotive Go | head -n 1
echo $?
```

Check codes with `./bin/vscan`. `go run` wraps the real status.

### Exit status

| Code | When |
| --- | --- |
| 0 | at least one match; the scanner or an apply run was interrupted; `--clear-cache` finished; stdout was closed; `--help`; `presets`; `alias` listed, saved, or deleted; `creds` listed, saved, or deleted; `--history` listed, including an empty history; `--response` confirmed at least one application |
| 1 | a one-shot search found nothing; `--response` confirmed nothing |
| 2 | bad arguments, an unknown flag, an unknown board, a broken query, a missing history entry, a broken alias, or auto-apply used in a binary built without it |

### Environment

| Variable | Values |
| --- | --- |
| `HH_TOKEN` | optional hh.ru API token. Empty: the public API is used |
| `SUPERJOB_API_KEY` | when set, SuperJob uses its API. When empty, SuperJob is read from public HTML |
| `XDG_CACHE_HOME` | replaces `~/.cache` |
| `XDG_DATA_HOME` | replaces `~/.local/share` |
| `XDG_CONFIG_HOME` | replaces `~/.config` |
| `PATH` | after `make install`, must contain `PREFIX/bin` |
| `PREFIX` | `make` variable, default `/usr/local`. Not read by the binary |
| `DESTDIR` | `make` variable, prepended to the install path. Not read by the binary |

The binary does not read a dotenv file.

### Files

| Path | Role |
| --- | --- |
| `~/.cache/vscan/seen/` | scanner memory. One URL per line. The file name is a hash of the expanded search and the sorted board list |
| `~/.local/share/vscan/history` | past queries, newest first |
| `~/.config/vscan/aliases` | `name=expression` lines |
| `~/.config/vscan/synonyms` | synonym groups for smart AND (`&\|`) |

```
remote = relocate, wfh, work from home
```

A synonym line replaces the whole built-in group for that word. The built-in groups are remote, senior, middle, and junior, including common translations. A missing synonyms file is not an error. Smart AND is the only mode that uses the file. Literal `&` does not.

### Diagnostics

Errors go to stderr as `vscan: ...` and exit 2 unless the table says otherwise.

| Message | Cause |
| --- | --- |
| `unknown flag --foo` | the flag is not in this section. Check the spelling |
| `the query must be one argument; quote it` | two positional words. Quote the query |
| `operator needs a word or a parenthesis after it` | `&&`, a trailing operator, a bare `!`, or a missing word |
| `--where: unknown place` | the value is not `sites` or `telegram` |
| `--max-age: want a whole number of days, at least 1` | the flag was `0` or not a number |
| `--sources applies to site boards` | `--sources` was set with `--where telegram` |
| doubled bar | `\|\|` has nothing between the two OR operators. The error is `\| needs a word or a parenthesis`. OR is one `\|` |
| `unknown platform "linkedin"` | the name is not in the board list. The error prints the list |
| `no platform selected` | `--platform` and `--sources` do not overlap |
| `--every only works with --scanner` | also `--listen` and `--webhook` |
| `--listen accepts only loopback` | the host is not `127.0.0.1` or `localhost` |
| `--headed only works with --response` | |
| `--response does not run a search` | a query, `--scanner`, or `--history` was also passed |
| `auto-apply is not included in this build` | `./bin/vscan` was built with `-tags noresponse`. Use `./bin/vscan-response` |
| `history is empty` | `--history` with nothing saved. Exit 0 |
| `no aliases` | `alias` with an empty file. Exit 0 |
| a board line and the others continue | that site returned an error or an empty anti-bot page. Exit code follows the matches from the boards that worked |

### Caveats

- macOS and Linux only. There is no Windows build.
- vscan does not bypass a login, a captcha, or an anti-bot page. HeadHunter often answers 403 from some networks. Djinni often returns an empty page. The error is on stderr and the other boards continue.
- `C++` and `C#` also match the word C, and `.NET` also matches the word net, because words are compared without punctuation.
- `Senior` in `Senior&Angular` is the whole word Senior. Lead, sr, and the Russian spellings apply only in smart mode: `Senior&|Angular`.
- Public feeds that ask for a credit, named in `--help` and not on each stdout line: [remoteok.com](https://remoteok.com), [weworkremotely.com](https://weworkremotely.com), [remotive.com](https://remotive.com), [jobicy.com](https://jobicy.com), [arbeitnow.com](https://www.arbeitnow.com). stdout is the job URL.

## 3. Presets

A language or framework name is a preset. `./bin/vscan Go` searches for Golang and Go. `./bin/vscan Angular` searches for Angular and AngularJS. The same names expand inside a larger query and inside `--profession`:

```bash
./bin/vscan Go
./bin/vscan angular
./bin/vscan 'C++'
./bin/vscan 'C#'
./bin/vscan 'Senior&|React&|Remote'
./bin/vscan --profession=TypeScript --platform='hh|habr'
./bin/vscan presets
```

`./bin/vscan presets` prints every name and the query it becomes. Letter case does not matter. A quoted name stays one word: `./bin/vscan '"Go"'` searches for Go and does not add Golang. An alias with the same name replaces the preset.

The matcher compares words, so `C++` and `C#` also match the word C, and `.NET` also matches the word net. The extra names (`CPP`, `CSharp`, `DotNet`) cover the spellings that are not punctuation.

### Languages

| Type any of | Search |
| --- | --- |
| `Go`, `Golang` | `Golang\|Go` |
| `JavaScript`, `JS` | `JavaScript\|JS` |
| `TypeScript`, `TS` | `TypeScript\|TS` |
| `Python`, `Py` | `Python` |
| `Java` | `Java` |
| `Kotlin` | `Kotlin` |
| `Swift` | `Swift` |
| `Rust` | `Rust` |
| `PHP` | `PHP` |
| `Ruby` | `Ruby` |
| `C` | `C` |
| `C++`, `CPP`, `CPlusPlus` | `C++\|CPP\|CPlusPlus` |
| `C#`, `CSharp` | `C#\|CSharp` |
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
| `F#`, `FSharp` | `F#\|FSharp` |
| `Objective-C`, `ObjC` | `Objective-C\|ObjC` |
| `Groovy` | `Groovy` |
| `MATLAB` | `MATLAB` |
| `Zig` | `Zig` |
| `HTML` | `HTML` |
| `CSS` | `CSS` |
| `Bash`, `Shell` | `Bash\|Shell` |
| `PowerShell` | `PowerShell` |

### Frameworks and platforms

| Type any of | Search |
| --- | --- |
| `Angular`, `AngularJS` | `Angular\|AngularJS` |
| `React`, `ReactJS`, `React.js` | `React\|ReactJS\|React.js` |
| `Vue`, `VueJS`, `Vue.js` | `Vue\|VueJS\|Vue.js` |
| `Svelte`, `SvelteKit` | `Svelte\|SvelteKit` |
| `Next`, `NextJS`, `Next.js` | `Next.js\|NextJS` |
| `Nuxt`, `NuxtJS`, `Nuxt.js` | `Nuxt\|NuxtJS` |
| `Node`, `NodeJS`, `Node.js` | `Node\|NodeJS\|Node.js` |
| `Express`, `ExpressJS` | `Express\|ExpressJS` |
| `Nest`, `NestJS`, `Nest.js` | `NestJS\|Nest.js` |
| `Django` | `Django` |
| `Flask` | `Flask` |
| `FastAPI` | `FastAPI` |
| `Spring`, `SpringBoot` | `Spring\|SpringBoot` |
| `Rails`, `RubyOnRails`, `RoR` | `Rails\|RubyOnRails\|RoR` |
| `Laravel` | `Laravel` |
| `Symfony` | `Symfony` |
| `Flutter` | `Flutter` |
| `ReactNative`, `React-Native` | `ReactNative\|React-Native` |
| `DotNet`, `.NET` | `DotNet\|.NET` |
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
| `Jetpack`, `Compose` | `Jetpack\|Compose` |
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
| `Senior\|Angular` | either word is enough |
| `Senior&\|Angular&\|Remote` | smart AND. Words are matched in the title, description, skills, and tags, in any order. `Angular` matches `AngularJS`. `Remote` matches `relocate` and `wfh` |
| `Senior&Angular&\|Remote` | `Senior` and `Angular` are literal, `Remote` uses synonyms |
| `(Senior\|Lead)&Angular` | parentheses. `&` and `&\|` bind tighter than `\|` |
| `"remote work"` | a phrase |
| `Angular&!React` | Angular, and not React. `!"full stack"` also drops `full-stack`. The single word `fullstack` needs its own `!fullstack` |
| `Angular&\|!Remote` | smart exclusion. `Remote` also drops `relocate` and `wfh` |
| `Angular&!(React\|Vue)` | drop either word. `!` binds tighter than `&` and `\|` |

The operator to the left of a word sets that word's mode. The first word takes the mode of the first operator. `!` uses that mode. A leading `!` is literal. One bare word is a literal search after presets expand, so `Go` becomes `Golang|Go` and `"Go"` stays the word Go. `!TypeScript` becomes `!(TypeScript|TS)`. Several bare words are an error, including `Angular !React`: write `Angular&!React`. OR is one `|`. Two bars in a row are an error. Quote the query in single quotes: the shell treats an unquoted `|` as a pipe, and `!` in double quotes is history expansion.

Excluded words are not sent to the boards. The board search stays the positive part (`Angular`), and `!` is applied to the text that comes back.

Field flags use the same operators and are combined with AND. An alias name expands in place of a word.

- `--company` matches the company name only. Hyphens are ignored, so `TBank` matches `T-Bank`.
- `--profession` matches the title and skills, not the full description.
- `--platform` picks boards. A vacancy has one source, so list several with `|`.

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
| `nofluff` | NoFluffJobs search API |
| `landing` | Landing.jobs |
| `muse` | The Muse, software engineering |
| `fourday` | 4dayweek.io |
| `jobspresso` | Jobspresso jobs RSS |
| `trudvsem` | Работа в России, open-data API |
| `hn` | Hacker News job stories |
| `python` | Python.org jobs RSS |
| `elixir` | Elixir Jobs RSS |
| `larajobs` | LaraJobs RSS |
| `golangprojects` | Golangprojects RSS |

```bash
./bin/vscan --platform='hh|habr|remotive|nofluff' --profession='Angular|TypeScript'
./bin/vscan --sources hh,habr,nofluff,landing 'Senior&Angular'
```

`--sources` and `--platform` use this table. `nofluff`, `landing`, `muse`, `fourday`, `jobspresso`, `trudvsem`, `hn`, `python`, `elixir`, `larajobs`, and `golangprojects` are valid in both. The Muse feed is the Software Engineering category. NoFluffJobs, The Muse, 4dayweek, Jobspresso, and Работа в России walk pages; with `--max-pages 0` each stops after five pages. Landing.jobs stops when its list ends, with the same five-page ceiling. Hacker News is one list of current job posts. A post with no link of its own uses the Hacker News item URL. Python.org, Elixir Jobs, LaraJobs, and Golangprojects are one RSS payload each. LaraJobs often leaves the description empty, so the title, company, and tags are what the query sees.

When both `--platform` and `--sources` are set, the result is their intersection. vscan does not bypass a login or a captcha. An empty or blocked page is an error on stderr.

`HH_TOKEN` is an optional hh.ru API token. `SUPERJOB_API_KEY`, when set, makes SuperJob use its API. Otherwise SuperJob is read from public HTML.

These feeds ask for a credit. stdout is the job URL. The feeds: [remoteok.com](https://remoteok.com), [weworkremotely.com](https://weworkremotely.com), [remotive.com](https://remotive.com), [jobicy.com](https://jobicy.com), [arbeitnow.com](https://www.arbeitnow.com).

## 6. Verbose output and JSON

The default line is a URL, so a pipe stays one job per line.

```bash
./bin/vscan --verbose --no-history --max-pages 1 --sources remotive 'Go|Golang'
./bin/vscan --json --no-history --max-pages 1 --sources jobicy 'Go|Golang'
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
  --platform=remoteok 'Go|Golang'
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
  --platform=remoteok 'Go|Golang'
```

Ctrl+C stops it with exit code 0. Day to day, use `--every 30` or `--every 60` and drop `--clear-cache` after the first run, so only new vacancies are reported.

Stdout still receives every emitted URL. [Telegram](#9-telegram-bot) is a pipe into your bot. [SSE and a webhook](#10-receive-scan-results) are the other two outputs. All three can run in the same command. A scanner pass also writes progress to stderr when stderr is a terminal.

## 9. Telegram bot

vscan does not talk to Telegram itself. The connection is a pipe: each new vacancy URL on stdout becomes one message from a bot you create.

### Create the bot

1. Open [@BotFather](https://t.me/BotFather) in Telegram and send `/newbot`.
2. Pick a name and a username that ends with `bot`.
3. Copy the token BotFather replies with. It looks like `123456:ABC-your-token`. Keep it out of the shell history if you can: a local file that is not committed is enough.

### Find your chat id

Open the new bot in Telegram and send it any message, for example `hi`. Then:

```bash
export TOKEN='123456:ABC-your-token'
curl -s "https://api.telegram.org/bot$TOKEN/getUpdates"
```

Use the number in `result[0].message.chat.id`. A private chat id is a positive number. A group id is negative. If `result` is empty, send the bot another message and run `getUpdates` again.

```bash
export CHAT='123456789'
```

### Send each new vacancy

Quote the query. `|` inside the quotes is OR. The `|` after the vscan command is the shell pipe into the loop.

```bash
export TOKEN='123456:ABC-your-token'
export CHAT='123456789'
./bin/vscan --clear-cache --scanner --every 1 --no-history --max-pages 1 \
  --platform=remoteok 'Go|Golang' | while read -r url; do
  curl -s -X POST "https://api.telegram.org/bot$TOKEN/sendMessage" \
    -d chat_id="$CHAT" --data-urlencode text="$url"
done
```

Leave that terminal open. Each new URL is one Telegram message.

`--clear-cache` makes the first pass send the vacancies that already exist. Without it, the first pass is silent and the bot gets a message only when a later pass finds a URL that was not seen before. After the first real run, drop `--clear-cache` so the same links are not sent again.

`--every 1` is for this check. For a watch you leave running, use `--every 30` or `--every 60`, and drop `--max-pages 1` when you want a fuller pass:

```bash
./bin/vscan --scanner --every 30 'Senior&|Angular|TypeScript' | while read -r url; do
  curl -s -X POST "https://api.telegram.org/bot$TOKEN/sendMessage" \
    -d chat_id="$CHAT" --data-urlencode text="$url"
done
```

Keep the default one-URL-per-line stdout. `--verbose` prints several lines per job and breaks `while read`.

### A message with the title

`--json` is still one line per vacancy, so `read` stays intact. `jq` builds the text:

```bash
./bin/vscan --clear-cache --json --scanner --every 1 --no-history --max-pages 1 \
  --platform=remoteok 'Go|Golang' | while read -r line; do
  text=$(printf '%s' "$line" | jq -r '"\(.title) — \(.company)\n\(.url)"')
  curl -s -X POST "https://api.telegram.org/bot$TOKEN/sendMessage" \
    -d chat_id="$CHAT" --data-urlencode text="$text"
done
```

`curl` must be installed. A failed `sendMessage` does not stop the scanner; the next URL is still read. Telegram answers `{"ok":false}` when the token or the chat id is wrong. Print that body while you are checking:

```bash
curl -s -X POST "https://api.telegram.org/bot$TOKEN/sendMessage" \
  -d chat_id="$CHAT" --data-urlencode text='vscan is connected'
```

`"ok":true` means the bot can reach that chat.

## 10. Receive scan results

The other two outputs are the SSE port (`--listen`) and an HTTP POST (`--webhook`). Telegram, above, is the stdout pipe. All three can run together.

### 10.1 SSE (`--listen`)

`--listen` opens a local HTTP server. The default address is `127.0.0.1:8787`. Only loopback is accepted (`127.0.0.1` or `localhost`). `0.0.0.0` is rejected.

Terminal 1, leave it running:

```bash
./bin/vscan --clear-cache --scanner --every 1 --no-history --max-pages 1 \
  --listen 127.0.0.1:8787 --platform=remoteok 'Go|Golang'
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

### 10.2 Webhook (`--webhook`)

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
  --platform=remoteok 'Go|Golang'
```

Terminal 1 prints one JSON object per vacancy. Point `--webhook` at your own service the same way. The URL has to be reachable from the machine running vscan.

### 10.3 Stdout, SSE, and a webhook

stdout, SSE, and the webhook can run in one process. Wrap the command with the pipe from [Telegram](#9-telegram-bot) when you also want a bot. A closed stdout pipe stops the whole scanner.

```bash
./bin/vscan --clear-cache --scanner --every 30 \
  --listen 127.0.0.1:8787 \
  --webhook http://127.0.0.1:8790/vacancy \
  'Senior&|Angular'
```

## 11. Aliases

A short name expands to an expression in the query and in flags. A quoted phrase is not expanded.

```bash
./bin/vscan alias myStack='Angular|Typescript|JavaScript|JS|TS|React'
./bin/vscan --profession=myStack --platform='hh|habr'
./bin/vscan alias
./bin/vscan alias --delete myStack
```

File `~/.config/vscan/aliases`, one line `myStack=Angular|Typescript|JavaScript|JS|TS|React`. A name starts with a letter or `_`, then letters, digits, and `_`. A cycle is an error. An alias whose name matches a preset replaces that preset.

## 12. History

```bash
./bin/vscan --history
./bin/vscan --history 3
./bin/vscan --no-history 'Senior&Angular'
```

File `~/.local/share/vscan/history` (`$XDG_DATA_HOME/vscan/history`). New queries go to the top. A repeat moves to the top and is not stored twice. The saved line is the query before alias expansion, so aliases expand again on `--history N`. `--history` without a number lists queries and does not search. `--clear-cache` does not delete this file.

## 13. Exit codes

| Code | When |
| --- | --- |
| 0 | at least one match; the scanner or an apply run was interrupted; `--clear-cache` finished; stdout was closed (`head`); `--help`; `presets`; `alias` or `creds` succeeded, including an empty list; `--history` listed queries, including an empty history; `--response` confirmed at least one application |
| 1 | a one-shot search found nothing, or `--response` confirmed nothing |
| 2 | bad arguments, an unknown flag, an unknown board, a broken query, a missing history entry, a broken alias, or auto-apply used in a binary built without it |

Run the built binary when you check these codes. `go run` wraps the real status.

## 14. Files and environment

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

## 15. Auto-apply

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

## Copyright

vscan is released under the MIT license. Copyright vscan contributors. See `LICENSE`.

