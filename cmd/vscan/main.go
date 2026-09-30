package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"vscan/internal/cli"
	"vscan/internal/emit"
	"vscan/internal/engine"
	"vscan/internal/httpx"
	"vscan/internal/provider"
	"vscan/internal/query"
	"vscan/internal/store"
)

func main() {
	signal.Ignore(syscall.SIGPIPE)
	opts, err := cli.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
		os.Exit(2)
	}
	if opts.Help {
		fmt.Print(cli.Help())
		os.Exit(0)
	}

	dirs := store.DefaultDirs()
	history := store.OpenHistory(dirs)
	if opts.History && opts.HistoryN == 0 && opts.Query == "" {
		lines, err := history.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
			os.Exit(2)
		}
		if len(lines) == 0 {
			fmt.Fprintln(os.Stderr, "vscan: история пуста")
			os.Exit(0)
		}
		for i, line := range lines {
			fmt.Printf("%d\t%s\n", i+1, line)
		}
		os.Exit(0)
	}
	if opts.History && opts.HistoryN > 0 {
		q, err := history.Get(opts.HistoryN)
		if err != nil {
			fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
			os.Exit(2)
		}
		opts.Query = q
	}
	if opts.Query == "" && !isTerminal(os.Stdin) {
		sc := bufio.NewScanner(os.Stdin)
		if sc.Scan() {
			opts.Query = strings.TrimSpace(sc.Text())
		}
		if err := sc.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
			os.Exit(2)
		}
	}
	if opts.Query == "" {
		fmt.Fprintln(os.Stderr, "vscan: нужен запрос (см. --help)")
		os.Exit(2)
	}

	expr, err := query.Parse(opts.Query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
		os.Exit(2)
	}
	if !opts.NoHistory {
		if err := history.Push(opts.Query); err != nil {
			fmt.Fprintf(os.Stderr, "vscan: история: %v\n", err)
		}
	}

	syn := query.BuiltinSynonyms()
	if err := syn.MergeFile(filepath.Join(dirs.Config, "synonyms")); err != nil {
		fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
		os.Exit(2)
	}

	sources := opts.Sources
	if len(sources) == 0 {
		sources = append([]string{}, provider.Names...)
	}
	client := httpx.New()
	boards, err := provider.Build(client, sources)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
		os.Exit(2)
	}

	var seen *store.Seen
	if opts.Scanner {
		key := append([]string{}, sources...)
		sort.Strings(key)
		seen, err = store.OpenSeen(dirs, opts.Query, key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
			os.Exit(2)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	stdout := emit.NewStdout(os.Stdout, opts.JSON)
	multi := emit.Multi{List: []emit.Emitter{stdout}}
	if opts.Listen != "" {
		sse := emit.NewSSE(opts.Listen)
		if err := sse.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "vscan: listen: %v\n", err)
			os.Exit(2)
		}
		defer func() {
			shut, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = sse.Shutdown(shut)
		}()
		multi.List = append(multi.List, sse)
		if isTerminal(os.Stderr) {
			fmt.Fprintf(os.Stderr, "vscan: события на http://%s/events\n", opts.Listen)
		}
	}
	if opts.Webhook != "" {
		multi.List = append(multi.List, &emit.Webhook{URL: opts.Webhook})
	}

	tty := isTerminal(os.Stderr)
	code := engine.Run(ctx, engine.Config{
		Expr:      expr,
		Syn:       syn,
		Providers: boards,
		Hints:     query.Hints(expr),
		MaxPages:  opts.MaxPages,
		Out:       multi,
		Seen:      seen,
		Logf: func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, "vscan: "+format+"\n", args...)
		},
		Progress: func(format string, args ...any) {
			if tty {
				fmt.Fprintf(os.Stderr, "vscan: "+format+"\n", args...)
			}
		},
	}, opts.Scanner, opts.Every)
	os.Exit(code)
}

func isTerminal(f *os.File) bool {
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}
