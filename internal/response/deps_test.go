package response

import (
	"testing"

	_ "github.com/chromedp/chromedp"
	_ "golang.org/x/term"
)

// The imports keep the optional-build dependencies in go.mod.
// They are compiled only into the test binary, not into `make build`.
func TestOptionalBuildDeps(t *testing.T) {}
