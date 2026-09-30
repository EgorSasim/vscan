//go:build response && !noresponse && darwin

package response

import "testing"

func TestParseAccount(t *testing.T) {
	sample := "attributes:\n    \"acct\"<blob>=\"me@example.com\"\n    \"svce\"<blob>=\"vscan/hh\"\n"
	if got := parseAccount(sample); got != "me@example.com" {
		t.Fatalf("got %q", got)
	}
	if parseAccount("no account") != "" {
		t.Fatal("expected empty")
	}
}
