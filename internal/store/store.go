// Package store keeps query history and the set of vacancy URLs already seen.
package store

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Dirs are the vscan roots. Empty fields fall back to XDG.
type Dirs struct {
	Cache  string
	Data   string
	Config string
}

// DefaultDirs follows XDG on macOS and Linux.
func DefaultDirs() Dirs {
	return Dirs{
		Cache:  xdg("XDG_CACHE_HOME", ".cache"),
		Data:   xdg("XDG_DATA_HOME", filepath.Join(".local", "share")),
		Config: xdg("XDG_CONFIG_HOME", ".config"),
	}
}

func xdg(env, fallback string) string {
	base := os.Getenv(env)
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			base = "."
		} else {
			base = filepath.Join(home, fallback)
		}
	}
	return filepath.Join(base, "vscan")
}

// History lists past queries, newest first.
type History struct {
	path string
}

func OpenHistory(d Dirs) *History {
	if d.Data == "" {
		d = DefaultDirs()
	}
	return &History{path: filepath.Join(d.Data, "history")}
}

func (h *History) List() ([]string, error) {
	b, err := os.ReadFile(h.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out, nil
}

// Get is 1-based, newest first.
func (h *History) Get(n int) (string, error) {
	lines, err := h.List()
	if err != nil {
		return "", err
	}
	if n < 1 || n > len(lines) {
		return "", fmt.Errorf("в истории нет записи %d", n)
	}
	return lines[n-1], nil
}

// Push moves query to the top. Repeats are not stored twice.
func (h *History) Push(query string) error {
	query = strings.TrimSpace(query)
	if query == "" || strings.Contains(query, "\n") {
		return fmt.Errorf("запрос для истории должен быть одной строкой")
	}
	lines, err := h.List()
	if err != nil {
		return err
	}
	next := []string{query}
	for _, line := range lines {
		if line != query {
			next = append(next, line)
		}
	}
	if err := os.MkdirAll(filepath.Dir(h.path), 0o755); err != nil {
		return err
	}
	body := strings.Join(next, "\n")
	if body != "" {
		body += "\n"
	}
	tmp := h.path + ".tmp"
	if err := os.WriteFile(tmp, []byte(body), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, h.path)
}

// Seen is the cache of vacancy URLs for one query and source set.
type Seen struct {
	path    string
	mu      sync.Mutex
	set     map[string]struct{}
	initial int
}

func OpenSeen(d Dirs, query string, sources []string) (*Seen, error) {
	if d.Cache == "" {
		d = DefaultDirs()
	}
	sum := sha256.Sum256([]byte(query + "\x00" + strings.Join(sources, ",")))
	path := filepath.Join(d.Cache, "seen", hex.EncodeToString(sum[:]))
	s := &Seen{path: path, set: map[string]struct{}{}}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		s.set[line] = struct{}{}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	s.initial = len(s.set)
	return s, nil
}

// InitialEmpty is true when the file had no URLs at open.
// Later Add calls do not change it: the scanner uses this for the baseline pass.
func (s *Seen) InitialEmpty() bool {
	return s.initial == 0
}

func (s *Seen) Has(url string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.set[url]
	return ok
}

// Add appends url immediately so a restart does not notify twice.
func (s *Seen) Add(url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.set[url]; ok {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, werr := f.WriteString(url + "\n")
	cerr := f.Close()
	if werr != nil {
		return werr
	}
	if cerr != nil {
		return cerr
	}
	s.set[url] = struct{}{}
	return nil
}
