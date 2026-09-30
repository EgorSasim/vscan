package store

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var aliasName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Aliases is the user's named query fragments.
type Aliases struct {
	path string
}

func OpenAliases(d Dirs) *Aliases {
	if d.Config == "" {
		d = DefaultDirs()
	}
	return &Aliases{path: filepath.Join(d.Config, "aliases")}
}

// Map returns name to body. Missing file is an empty map.
func (a *Aliases) Map() (map[string]string, error) {
	lines, err := a.read()
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(lines))
	for _, line := range lines {
		name, body, _ := strings.Cut(line, "=")
		out[name] = body
	}
	return out, nil
}

// List returns definitions in file order, as "name=body".
func (a *Aliases) List() ([]string, error) {
	return a.read()
}

// Set creates or replaces an alias.
func (a *Aliases) Set(name, body string) error {
	name = strings.TrimSpace(name)
	body = strings.TrimSpace(body)
	if !aliasName.MatchString(name) {
		return fmt.Errorf("alias name must start with a letter or _, then letters, digits, or _")
	}
	if body == "" {
		return fmt.Errorf("empty alias body %s", name)
	}
	if strings.Contains(body, "\n") {
		return fmt.Errorf("alias body must be one line")
	}
	lines, err := a.read()
	if err != nil {
		return err
	}
	row := name + "=" + body
	replaced := false
	for i, line := range lines {
		cur, _, _ := strings.Cut(line, "=")
		if cur == name {
			lines[i] = row
			replaced = true
			break
		}
	}
	if !replaced {
		lines = append(lines, row)
	}
	return a.write(lines)
}

// Delete removes an alias.
func (a *Aliases) Delete(name string) error {
	lines, err := a.read()
	if err != nil {
		return err
	}
	next := lines[:0]
	found := false
	for _, line := range lines {
		cur, _, _ := strings.Cut(line, "=")
		if cur == name {
			found = true
			continue
		}
		next = append(next, line)
	}
	if !found {
		return fmt.Errorf("no alias %s", name)
	}
	return a.write(next)
}

func (a *Aliases) read() ([]string, error) {
	b, err := os.ReadFile(a.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for n, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, body, ok := strings.Cut(line, "=")
		name = strings.TrimSpace(name)
		body = strings.TrimSpace(body)
		if !ok || !aliasName.MatchString(name) || body == "" {
			return nil, fmt.Errorf("%s:%d: expected a line like myStack=Angular|TS", a.path, n+1)
		}
		out = append(out, name+"="+body)
	}
	return out, nil
}

func (a *Aliases) write(lines []string) error {
	if err := os.MkdirAll(filepath.Dir(a.path), 0o755); err != nil {
		return err
	}
	body := ""
	if len(lines) > 0 {
		body = strings.Join(lines, "\n") + "\n"
	}
	tmp := a.path + ".tmp"
	if err := os.WriteFile(tmp, []byte(body), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, a.path)
}
