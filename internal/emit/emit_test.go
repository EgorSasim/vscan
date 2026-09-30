package emit

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStdoutFlushesBeforeClose(t *testing.T) {
	r, w := io.Pipe()
	out := NewStdout(w, false, false)
	done := make(chan string, 1)
	go func() {
		buf := make([]byte, 64)
		n, err := r.Read(buf)
		if err != nil {
			done <- err.Error()
			return
		}
		done <- string(buf[:n])
	}()
	if err := out.Emit(Hit{URL: "https://example.com/a"}); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-done:
		if got != "https://example.com/a\n" {
			t.Fatalf("got %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("line was not flushed")
	}
}

func TestStdoutJSON(t *testing.T) {
	var b strings.Builder
	out := NewStdout(&b, true, false)
	if err := out.Emit(Hit{URL: "https://example.com/a", Title: "A & B", Company: "C", Source: "hh"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), `"url":"https://example.com/a"`) || !strings.Contains(b.String(), "A & B") {
		t.Fatalf("json = %s", b.String())
	}
}

func TestStdoutVerbose(t *testing.T) {
	var b strings.Builder
	out := NewStdout(&b, false, true)
	err := out.Emit(Hit{
		URL: "https://example.com/a", Title: "Go", Company: "Acme", Source: "habr",
		Remote: "yes", Age: "2 days",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := b.String()
	if !strings.Contains(got, "https://example.com/a\n") || !strings.Contains(got, "remote: yes\n") || !strings.Contains(got, "age: 2 days\n") {
		t.Fatalf("verbose = %q", got)
	}
	if strings.Contains(got, "salary:") {
		t.Fatal("empty salary")
	}
}

func TestPipeError(t *testing.T) {
	r, w := io.Pipe()
	out := NewStdout(w, false, false)
	_ = r.Close()
	err := out.Emit(Hit{URL: "https://example.com/a"})
	if err != ErrPipe {
		t.Fatalf("err = %v", err)
	}
}

func TestWebhook(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = string(b)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	hook := &Webhook{URL: srv.URL, Client: srv.Client()}
	if err := hook.Emit(Hit{URL: "https://example.com/a", Source: "hh"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "https://example.com/a") {
		t.Fatalf("body = %s", got)
	}
}

func TestLoopback(t *testing.T) {
	if !Loopback("127.0.0.1:8787") || !Loopback("[::1]:8787") || !Loopback("localhost:8787") {
		t.Fatal("loopback rejected")
	}
	if Loopback("0.0.0.0:8787") || Loopback("example.com:8787") {
		t.Fatal("non-loopback accepted")
	}
}
