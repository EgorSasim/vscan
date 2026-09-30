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
	"vscan/internal/query"
	"vscan/internal/sources"
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
	if opts.ClearCache {
		n, err := store.ClearSeen(dirs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
			os.Exit(2)
		}
		if !opts.HasSearch() && !opts.Scanner && !opts.History {
			fmt.Printf("cleared %d remembered links\n", n)
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "vscan: cleared %d remembered links\n", n)
	}
	history := store.OpenHistory(dirs)
	aliases := store.OpenAliases(dirs)
	if opts.AliasList || opts.AliasName != "" || opts.AliasDel != "" {
		os.Exit(runAlias(aliases, opts))
	}
	if opts.History && opts.HistoryN == 0 && !opts.HasSearch() {
		lines, err := history.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
			os.Exit(2)
		}
		if len(lines) == 0 {
			fmt.Fprintln(os.Stderr, "vscan: history is empty")
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
		args, err := cli.Split(q)
		if err != nil {
			fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
			os.Exit(2)
		}
		saved, err := cli.Parse(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
			os.Exit(2)
		}
		opts.Query = saved.Query
		opts.Company = saved.Company
		opts.Profession = saved.Profession
		opts.Platform = saved.Platform
	}
	if !opts.HasSearch() && !isTerminal(os.Stdin) {
		sc := bufio.NewScanner(os.Stdin)
		if sc.Scan() {
			opts.Query = strings.TrimSpace(sc.Text())
		}
		if err := sc.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
			os.Exit(2)
		}
	}
	if !opts.HasSearch() {
		fmt.Fprintln(os.Stderr, "vscan: a query or --company, --profession, or --platform is required (see --help)")
		os.Exit(2)
	}

	savedSearch := opts.SearchRecord()
	aliasMap, err := aliases.Map()
	if err != nil {
		fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
		os.Exit(2)
	}
	opts.Query, err = query.Expand(opts.Query, aliasMap)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
		os.Exit(2)
	}
	opts.Company, err = query.Expand(opts.Company, aliasMap)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
		os.Exit(2)
	}
	opts.Profession, err = query.Expand(opts.Profession, aliasMap)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
		os.Exit(2)
	}
	opts.Platform, err = query.Expand(opts.Platform, aliasMap)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
		os.Exit(2)
	}

	expr, err := parseOptional(opts.Query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
		os.Exit(2)
	}
	companyExpr, err := parseOptional(opts.Company)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vscan: --company: %v\n", err)
		os.Exit(2)
	}
	professionExpr, err := parseOptional(opts.Profession)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vscan: --profession: %v\n", err)
		os.Exit(2)
	}
	platformExpr, err := parseOptional(opts.Platform)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vscan: --platform: %v\n", err)
		os.Exit(2)
	}
	if !opts.NoHistory {
		if err := history.Push(savedSearch); err != nil {
			fmt.Fprintf(os.Stderr, "vscan: history: %v\n", err)
		}
	}

	syn := query.BuiltinSynonyms()
	if err := syn.MergeFile(filepath.Join(dirs.Config, "synonyms")); err != nil {
		fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
		os.Exit(2)
	}

	names := opts.Sources
	if len(names) == 0 {
		names = append([]string{}, sources.Names...)
	}
	if platformExpr != nil {
		names, err = selectPlatforms(names, platformExpr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
			os.Exit(2)
		}
	}
	client := httpx.New()
	boards, err := sources.Build(client, names)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
		os.Exit(2)
	}

	var seen *store.Seen
	if opts.Scanner {
		key := append([]string{}, names...)
		sort.Strings(key)
		seen, err = store.OpenSeen(dirs, opts.SearchRecord(), key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
			os.Exit(2)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	stdout := emit.NewStdout(os.Stdout, opts.JSON, opts.Verbose)
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
			fmt.Fprintf(os.Stderr, "vscan: events at http://%s/events\n", opts.Listen)
		}
	}
	if opts.Webhook != "" {
		multi.List = append(multi.List, &emit.Webhook{URL: opts.Webhook})
	}

	tty := isTerminal(os.Stderr)
	code := engine.Run(ctx, engine.Config{
		Expr:       expr,
		Company:    companyExpr,
		Profession: professionExpr,
		Platform:   platformExpr,
		Syn:        syn,
		Providers:  boards,
		Hints:      query.CollectHints(expr, professionExpr, companyExpr),
		MaxPages:   opts.MaxPages,
		Out:        multi,
		Seen:       seen,
		Logf: func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, "vscan: "+format+"\n", args...)
		},
		Progress: func(format string, args ...any) {
			if tty {
				fmt.Fprintf(os.Stderr, "vscan: "+format+"\n", args...)
			}
		},
		Replay: opts.ClearCache,
	}, opts.Scanner, opts.Every)
	os.Exit(code)
}

func parseOptional(q string) (query.Expr, error) {
	if strings.TrimSpace(q) == "" {
		return nil, nil
	}
	return query.Parse(q)
}

func selectPlatforms(selected []string, expr query.Expr) ([]string, error) {
	allowed := map[string]struct{}{}
	for _, raw := range query.Terms(expr) {
		id, ok := sources.Canonical(raw)
		if !ok {
			return nil, fmt.Errorf("unknown platform %q (%s)", raw, strings.Join(sources.Names, ", "))
		}
		allowed[id] = struct{}{}
	}
	var out []string
	seen := map[string]struct{}{}
	for _, name := range selected {
		id, ok := sources.Canonical(name)
		if !ok {
			return nil, fmt.Errorf("unknown platform %q (%s)", name, strings.Join(sources.Names, ", "))
		}
		if _, ok := allowed[id]; !ok {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no platform selected")
	}
	return out, nil
}

func runAlias(aliases *store.Aliases, opts cli.Options) int {
	switch {
	case opts.AliasList:
		lines, err := aliases.List()
		if err != nil {
			fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
			return 2
		}
		if len(lines) == 0 {
			fmt.Fprintln(os.Stderr, "vscan: no aliases")
			return 0
		}
		for _, line := range lines {
			fmt.Println(line)
		}
		return 0
	case opts.AliasDel != "":
		if err := aliases.Delete(opts.AliasDel); err != nil {
			fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
			return 2
		}
		return 0
	default:
		table, err := aliases.Map()
		if err != nil {
			fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
			return 2
		}
		table[opts.AliasName] = opts.AliasValue
		expanded, err := query.Expand(opts.AliasValue, table)
		if err != nil {
			fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
			return 2
		}
		if _, err := query.Parse(expanded); err != nil {
			fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
			return 2
		}
		if err := aliases.Set(opts.AliasName, opts.AliasValue); err != nil {
			fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
			return 2
		}
		fmt.Printf("%s=%s\n", opts.AliasName, opts.AliasValue)
		return 0
	}
}

func isTerminal(f *os.File) bool {
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}
