//go:build response && !noresponse

package response

import (
	"context"
	"errors"
	"testing"
)

type memStore struct {
	m map[string][2]string
}

func (s memStore) Set(platform, login, password string) error {
	if s.m == nil {
		return errors.New("nil")
	}
	s.m[platform] = [2]string{login, password}
	return nil
}

func (s memStore) Get(platform string) (string, string, error) {
	pair, ok := s.m[platform]
	if !ok {
		return "", "", errors.New("no saved login for " + platform)
	}
	return pair[0], pair[1], nil
}

func (s memStore) Delete(platform string) error { delete(s.m, platform); return nil }

func (s memStore) List() ([]string, error) {
	var out []string
	for name := range s.m {
		out = append(out, name)
	}
	return out, nil
}

type scripted struct {
	errFor map[string]error
	calls  []string
}

func (s *scripted) Apply(_ context.Context, platform, vacancyURL, login, password string) error {
	s.calls = append(s.calls, platform+" "+login)
	if password == "" {
		return errors.New("empty password")
	}
	if err, ok := s.errFor[vacancyURL]; ok {
		return err
	}
	return nil
}

func (s *scripted) Close() {}

func TestApplyAllSkipsAndContinues(t *testing.T) {
	links := ParseLinks("https://hh.ru/vacancy/1\x1fhttps://hh.ru/vacancy/2\nhttps://remoteok.com/remote-jobs/9\nnot-a-url")
	br := &scripted{errFor: map[string]error{
		"https://hh.ru/vacancy/2": errors.New("captcha or a confirmation code is required"),
	}}
	var logs []string
	n := applyAll(context.Background(), links, memStore{m: map[string][2]string{
		"hh": {"me@example.com", "secret"},
	}}, br, func(format string, args ...any) {
		logs = append(logs, format)
	})
	if n != 1 {
		t.Fatalf("applied %d", n)
	}
	if len(br.calls) != 2 {
		t.Fatalf("calls %#v", br.calls)
	}
	if len(logs) < 3 {
		t.Fatalf("logs %#v", logs)
	}
}
