//go:build response && !noresponse && darwin

package response

import (
	"fmt"
	"os/exec"
	"strings"
)

type osStore struct{}

func openOSStore() secretStore { return osStore{} }

func service(platform string) string { return "vscan/" + platform }

func (osStore) Set(platform, login, password string) error {
	cmd := exec.Command("security", "add-generic-password",
		"-U", "-s", service(platform), "-a", login, "-w", password)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("keychain update failed: %s", firstLine(out))
	}
	return nil
}

func (osStore) Get(platform string) (string, string, error) {
	meta, err := exec.Command("security", "find-generic-password", "-s", service(platform)).CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("no saved login for %s", platform)
	}
	login := parseAccount(string(meta))
	if login == "" {
		return "", "", fmt.Errorf("keychain entry for %s has no account", platform)
	}
	pw, err := exec.Command("security", "find-generic-password", "-s", service(platform), "-w").Output()
	if err != nil {
		return "", "", fmt.Errorf("keychain read failed for %s", platform)
	}
	password := strings.TrimRight(string(pw), "\r\n")
	if password == "" {
		return "", "", fmt.Errorf("keychain entry for %s has an empty password", platform)
	}
	return login, password, nil
}

func (osStore) Delete(platform string) error {
	if err := exec.Command("security", "delete-generic-password", "-s", service(platform)).Run(); err != nil {
		return fmt.Errorf("no saved login for %s", platform)
	}
	return nil
}

func (osStore) List() ([]string, error) {
	var out []string
	for _, id := range ApplyBoards {
		if err := exec.Command("security", "find-generic-password", "-s", service(id)).Run(); err == nil {
			out = append(out, id)
		}
	}
	return out, nil
}

func parseAccount(output string) string {
	const mark = `"acct"<blob>="`
	i := strings.Index(output, mark)
	if i < 0 {
		return ""
	}
	rest := output[i+len(mark):]
	j := strings.IndexByte(rest, '"')
	if j < 0 {
		return ""
	}
	return rest[:j]
}

func firstLine(b []byte) string {
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "security failed"
	}
	line, _, _ := strings.Cut(s, "\n")
	if len(line) > 200 {
		line = line[:200]
	}
	return line
}
