//go:build response && !noresponse

package response

import "context"

type secretStore interface {
	Set(platform, login, password string) error
	Get(platform string) (login, password string, err error)
	Delete(platform string) error
	List() ([]string, error)
}

type browser interface {
	Apply(ctx context.Context, platform, vacancyURL, login, password string) error
	Close()
}

func applyAll(ctx context.Context, links []Link, store secretStore, br browser, logf func(string, ...any)) int {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	applied := 0
	for _, link := range links {
		if ctx.Err() != nil {
			return applied
		}
		if link.Problem != "" {
			logf("skip %s: %s", link.Raw, link.Problem)
			continue
		}
		login, password, err := store.Get(link.Board)
		if err != nil {
			logf("skip %s: %v", link.Raw, err)
			continue
		}
		if err := br.Apply(ctx, link.Board, link.Raw, login, password); err != nil {
			if ctx.Err() != nil {
				return applied
			}
			logf("skip %s: %v", link.Raw, err)
			continue
		}
		logf("applied %s", link.Raw)
		applied++
	}
	return applied
}
