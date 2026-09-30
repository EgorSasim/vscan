//go:build response && !noresponse && linux

package response

import (
	"fmt"
	"os/exec"
	"strings"
)

type osStore struct{}

func openOSStore() secretStore { return osStore{} }

func (osStore) Set(platform, login, password string) error {
	if err := secretPut(platform, "user", login); err != nil {
		return err
	}
	return secretPut(platform, "pass", password)
}

func (osStore) Get(platform string) (string, string, error) {
	login, err := secretLookup(platform, "user")
	if err != nil {
		return "", "", fmt.Errorf("no saved login for %s", platform)
	}
	password, err := secretLookup(platform, "pass")
	if err != nil {
		return "", "", fmt.Errorf("no saved login for %s", platform)
	}
	return login, password, nil
}

func (osStore) Delete(platform string) error {
	errUser := secretClear(platform, "user")
	errPass := secretClear(platform, "pass")
	if errUser != nil && errPass != nil {
		return fmt.Errorf("no saved login for %s", platform)
	}
	return nil
}

func (osStore) List() ([]string, error) {
	var out []string
	for _, id := range ApplyBoards {
		if _, err := secretLookup(id, "user"); err == nil {
			out = append(out, id)
		}
	}
	return out, nil
}

func secretPut(platform, kind, secret string) error {
	cmd := exec.Command("secret-tool", "store", "--label", "vscan "+platform+" "+kind,
		"service", "vscan", "platform", platform, "kind", kind)
	cmd.Stdin = strings.NewReader(secret)
	if out, err := cmd.CombinedOutput(); err != nil {
		if len(out) == 0 {
			return fmt.Errorf("secret-tool failed; install libsecret-tools")
		}
		return fmt.Errorf("secret-tool failed")
	}
	return nil
}

func secretLookup(platform, kind string) (string, error) {
	out, err := exec.Command("secret-tool", "lookup",
		"service", "vscan", "platform", platform, "kind", kind).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

func secretClear(platform, kind string) error {
	return exec.Command("secret-tool", "clear",
		"service", "vscan", "platform", platform, "kind", kind).Run()
}
