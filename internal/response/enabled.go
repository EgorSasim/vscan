//go:build response && !noresponse

package response

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// Enabled is true only in a binary built with -tags response.
func Enabled() bool { return true }

// Creds stores, lists, or deletes a board login in the OS keychain.
func Creds(action, platform string) int {
	switch action {
	case "list":
		return credsList()
	case "set":
		return credsSet(platform)
	case "delete":
		return credsDelete(platform)
	default:
		fmt.Fprintln(os.Stderr, "vscan: vscan creds [set|delete] PLATFORM")
		return 2
	}
}

func credsList() int {
	names, err := openOSStore().List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
		return 2
	}
	if len(names) == 0 {
		fmt.Fprintln(os.Stderr, "vscan: no saved logins")
		return 0
	}
	for _, name := range names {
		fmt.Println(name)
	}
	return 0
}

func credsSet(platform string) int {
	if !KnownBoard(platform) {
		fmt.Fprintf(os.Stderr, "vscan: %v\n", unknownBoard(platform))
		return 2
	}
	login, password, err := promptCreds()
	if err != nil {
		fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
		return 2
	}
	if err := openOSStore().Set(platform, login, password); err != nil {
		fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
		return 2
	}
	fmt.Printf("saved login for %s\n", platform)
	return 0
}

func credsDelete(platform string) int {
	if !KnownBoard(platform) {
		fmt.Fprintf(os.Stderr, "vscan: %v\n", unknownBoard(platform))
		return 2
	}
	if err := openOSStore().Delete(platform); err != nil {
		fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
		return 2
	}
	fmt.Printf("deleted login for %s\n", platform)
	return 0
}

// Run applies to every vacancy URL. Details go to stderr.
// Stdout is one line: applied N.
func Run(ctx context.Context, payload string, stdin io.Reader, stdinTTY, headed bool) int {
	text := payload
	if !stdinTTY && stdin != nil {
		b, err := io.ReadAll(stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
			return 2
		}
		if strings.TrimSpace(string(b)) != "" {
			if text != "" {
				text += "\n"
			}
			text += string(b)
		}
	}
	links := ParseLinks(text)
	if len(links) == 0 {
		fmt.Fprintln(os.Stderr, "vscan: --response needs vacancy URLs (argument or stdin)")
		return 2
	}
	store := openOSStore()
	br, err := newBrowser(ctx, headed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vscan: %v\n", err)
		return 2
	}
	defer br.Close()
	n := applyAll(ctx, links, store, br, func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, "vscan: "+format+"\n", args...)
	})
	if ctx.Err() != nil {
		fmt.Printf("applied %d\n", n)
		return 0
	}
	fmt.Printf("applied %d\n", n)
	if n == 0 {
		return 1
	}
	return 0
}
