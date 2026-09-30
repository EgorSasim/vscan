//go:build response && !noresponse

package response

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
)

type loginSpec struct {
	URL    string
	User   string
	Pass   string
	Submit string
}

var logins = map[string]loginSpec{
	"hh": {
		URL:    "https://hh.ru/account/login?role=applicant",
		User:   `input[data-qa="login-input-username"]`,
		Pass:   `input[data-qa="login-input-password"]`,
		Submit: `button[data-qa="account-login-submit"]`,
	},
	"habr": {
		URL:    "https://career.habr.com/users/sign_in",
		User:   `input[type="email"], input[name="email"]`,
		Pass:   `input[type="password"]`,
		Submit: `button[type="submit"], input[type="submit"]`,
	},
	"superjob": {
		URL:    "https://www.superjob.ru/auth/login/",
		User:   `input[name="login"], input[type="email"]`,
		Pass:   `input[type="password"]`,
		Submit: `button[type="submit"], input[type="submit"]`,
	},
	"djinni": {
		URL:    "https://djinni.co/login",
		User:   `input[name="email"], input[type="email"]`,
		Pass:   `input[type="password"]`,
		Submit: `button[type="submit"], input[type="submit"]`,
	},
	"getmatch": {
		URL:    "https://getmatch.ru/login",
		User:   `input[type="email"], input[name="email"]`,
		Pass:   `input[type="password"]`,
		Submit: `button[type="submit"]`,
	},
	"geekjob": {
		URL:    "https://geekjob.ru/login",
		User:   `input[type="email"], input[name="email"]`,
		Pass:   `input[type="password"]`,
		Submit: `button[type="submit"], input[type="submit"]`,
	},
}

type chromeBrowser struct {
	cancel  context.CancelFunc
	ctx     context.Context
	ready   map[string]struct{}
	blocked map[string]error
}

func newBrowser(ctx context.Context, headed bool) (browser, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", !headed),
	)
	alloc, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	tab, cancelTab := chromedp.NewContext(alloc)
	return &chromeBrowser{
		cancel: func() {
			cancelTab()
			cancelAlloc()
		},
		ctx:     tab,
		ready:   map[string]struct{}{},
		blocked: map[string]error{},
	}, nil
}

func (c *chromeBrowser) Close() {
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *chromeBrowser) Apply(parent context.Context, platform, vacancyURL, login, password string) error {
	if err, ok := c.blocked[platform]; ok {
		return err
	}
	ctx, cancel := context.WithTimeout(c.ctx, 50*time.Second)
	go func() {
		select {
		case <-parent.Done():
			cancel()
		case <-ctx.Done():
		}
	}()
	defer cancel()
	if _, ok := c.ready[platform]; !ok {
		if err := c.login(ctx, platform, login, password); err != nil {
			c.blocked[platform] = err
			return err
		}
		c.ready[platform] = struct{}{}
	}
	return c.click(ctx, vacancyURL)
}

func (c *chromeBrowser) login(ctx context.Context, platform, login, password string) error {
	spec, ok := logins[platform]
	if !ok {
		return fmt.Errorf("no login page for %s", platform)
	}
	if err := chromedp.Run(ctx,
		chromedp.Navigate(spec.URL),
		chromedp.WaitReady("body", chromedp.ByQuery),
	); err != nil {
		return err
	}
	if err := c.barrier(ctx); err != nil {
		return err
	}
	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(spec.Pass, chromedp.ByQuery),
		chromedp.WaitVisible(spec.User, chromedp.ByQuery),
		chromedp.SendKeys(spec.User, login, chromedp.ByQuery),
		chromedp.SendKeys(spec.Pass, password, chromedp.ByQuery),
		chromedp.Click(spec.Submit, chromedp.ByQuery),
	); err != nil {
		return fmt.Errorf("login form: %w", err)
	}
	if err := chromedp.Run(ctx, chromedp.Sleep(2*time.Second)); err != nil {
		return err
	}
	if err := c.barrier(ctx); err != nil {
		return err
	}
	var still bool
	if err := chromedp.Run(ctx, chromedp.Evaluate(passwordStillVisibleJS, &still)); err != nil {
		return err
	}
	if still {
		return errors.New("login failed")
	}
	return nil
}

func (c *chromeBrowser) click(ctx context.Context, vacancyURL string) error {
	if err := chromedp.Run(ctx,
		chromedp.Navigate(vacancyURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(1500*time.Millisecond),
	); err != nil {
		return err
	}
	if err := c.barrier(ctx); err != nil {
		return err
	}
	var status string
	if err := chromedp.Run(ctx, chromedp.Evaluate(clickApplyJS, &status)); err != nil {
		return err
	}
	switch status {
	case "already":
		return errors.New("already applied")
	case "missing":
		return errors.New("apply button was not found")
	case "clicked":
	default:
		return fmt.Errorf("apply button: %s", status)
	}
	if err := chromedp.Run(ctx, chromedp.Sleep(1500*time.Millisecond)); err != nil {
		return err
	}
	if err := c.barrier(ctx); err != nil {
		return err
	}
	var follow string
	if err := chromedp.Run(ctx, chromedp.Evaluate(afterClickJS, &follow)); err != nil {
		return err
	}
	switch follow {
	case "questions":
		return errors.New("the form asks extra questions")
	case "applied", "confirmed":
		return nil
	case "uncertain":
		return errors.New("could not confirm the application")
	default:
		return fmt.Errorf("apply: %s", follow)
	}
}

func (c *chromeBrowser) barrier(ctx context.Context) error {
	var kind string
	if err := chromedp.Run(ctx, chromedp.Evaluate(barrierJS, &kind)); err != nil {
		return err
	}
	switch kind {
	case "", "ok":
		return nil
	case "captcha":
		return errors.New("captcha or a confirmation code is required")
	default:
		return errors.New(kind)
	}
}

const passwordStillVisibleJS = `(function () {
  const el = document.querySelector('input[type="password"]');
  if (!el) return false;
  const style = window.getComputedStyle(el);
  return style.display !== 'none' && style.visibility !== 'hidden';
})()`

const barrierJS = `(function () {
  if (document.querySelector('iframe[src*="captcha"], .g-recaptcha, .h-captcha, iframe[src*="hcaptcha"], iframe[src*="recaptcha"]')) {
    return 'captcha';
  }
  if (document.querySelector('input[autocomplete="one-time-code"], input[name*="otp" i]')) {
    return 'captcha';
  }
  const text = (document.body && document.body.innerText || '').toLowerCase();
  if (text.includes('подтвердите, что вы не робот') || text.includes('я не робот')) {
    return 'captcha';
  }
  return 'ok';
})()`

const clickApplyJS = `(function () {
  const els = Array.from(document.querySelectorAll('button, a, input[type="submit"]'));
  for (const el of els) {
    const text = (el.innerText || el.value || '').replace(/\s+/g, ' ').trim().toLowerCase();
    if (!text) continue;
    if (text.includes('вы откликнулись') || text.includes('already applied') || text === 'откликнулись') {
      return 'already';
    }
  }
  const labels = ['откликнуться', 'откликнуться на вакансию', 'apply', 'apply now', 'respond'];
  for (const el of els) {
    const text = (el.innerText || el.value || '').replace(/\s+/g, ' ').trim().toLowerCase();
    if (labels.some(function (label) { return text === label || text.indexOf(label) === 0; })) {
      el.click();
      return 'clicked';
    }
  }
  return 'missing';
})()`

const afterClickJS = `(function () {
  const text = (document.body && document.body.innerText || '').toLowerCase();
  if (text.includes('отклик отправлен') || text.includes('вы откликнулись') || text.includes('резюме отправлено') || text.includes('you have applied') || text.includes('application has been sent')) {
    return 'applied';
  }
  const dialog = document.querySelector('[role="dialog"], [class*="modal"], [class*="popup"]');
  if (!dialog) return 'uncertain';
  const fields = Array.from(dialog.querySelectorAll('textarea, input, select')).filter(function (el) {
    if (el.disabled || el.type === 'hidden' || el.type === 'checkbox' || el.type === 'radio') return false;
    const blob = ((el.name || '') + ' ' + (el.getAttribute('aria-label') || '') + ' ' + (el.id || '')).toLowerCase();
    if (blob.indexOf('resume') >= 0 || blob.indexOf('резюме') >= 0) return false;
    return el.required && !el.value;
  });
  if (fields.length) return 'questions';
  const send = Array.from(dialog.querySelectorAll('button, input[type="submit"]')).find(function (el) {
    const label = (el.innerText || el.value || '').toLowerCase();
    return label.indexOf('откликнуться') >= 0 || label.indexOf('отправить') >= 0 || label.indexOf('apply') >= 0;
  });
  if (send) {
    send.click();
    return 'confirmed';
  }
  return 'uncertain';
})()`
