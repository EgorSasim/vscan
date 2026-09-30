//go:build !response || noresponse

package response

import (
	"context"
	"fmt"
	"io"
	"os"
)

// NotIncluded is the error text of a binary built without -tags response.
const NotIncluded = "auto-apply is not included in this build; compile with -tags response"

// Enabled is false in the default build. The VPS binary uses this file.
func Enabled() bool { return false }

// Creds refuses to store or show logins.
func Creds(action, platform string) int {
	fmt.Fprintln(os.Stderr, "vscan: "+NotIncluded)
	return 2
}

// Run refuses to apply.
func Run(ctx context.Context, payload string, stdin io.Reader, stdinTTY, headed bool) int {
	fmt.Fprintln(os.Stderr, "vscan: "+NotIncluded)
	return 2
}
