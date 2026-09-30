// Package emit writes vacancy hits to stdout, an SSE port, and a webhook.
package emit

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"syscall"
	"time"
)

// Hit is one matching vacancy.
type Hit struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Company string `json:"company"`
	Source  string `json:"source"`
}

// Emitter receives hits. Emit must be safe for concurrent use.
type Emitter interface {
	Emit(Hit) error
}

// ErrPipe means the stdout reader went away.
var ErrPipe = errors.New("pipe closed")

func isPipe(err error) bool {
	return errors.Is(err, syscall.EPIPE) || errors.Is(err, io.ErrClosedPipe)
}

// Stdout writes one URL or one JSON object per line and flushes immediately.
type Stdout struct {
	w    *bufio.Writer
	json bool
	mu   sync.Mutex
}

func NewStdout(w io.Writer, asJSON bool) *Stdout {
	return &Stdout{w: bufio.NewWriter(w), json: asJSON}
}

func (s *Stdout) Emit(h Hit) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var err error
	if s.json {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err = enc.Encode(h); err != nil {
			return err
		}
		_, err = s.w.Write(bytes.TrimRight(buf.Bytes(), "\n"))
		if err == nil {
			err = s.w.WriteByte('\n')
		}
	} else {
		_, err = s.w.WriteString(h.URL + "\n")
	}
	if err == nil {
		err = s.w.Flush()
	}
	if isPipe(err) {
		return ErrPipe
	}
	return err
}

// Multi fans a hit out to every child. A closed pipe stops the process.
// Other child errors are returned after every child has been tried.
type Multi struct {
	List []Emitter
}

func (m Multi) Emit(h Hit) error {
	var first error
	for _, e := range m.List {
		if e == nil {
			continue
		}
		if err := e.Emit(h); err != nil {
			if errors.Is(err, ErrPipe) {
				return err
			}
			if first == nil {
				first = err
			}
		}
	}
	return first
}

// Webhook POSTs each hit as JSON. Failures are retried twice.
type Webhook struct {
	URL    string
	Client *http.Client
}

func (w *Webhook) Emit(h Hit) error {
	if w.URL == "" {
		return nil
	}
	body, err := marshal(h)
	if err != nil {
		return err
	}
	client := w.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest(http.MethodPost, w.URL, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
			err = fmt.Errorf("webhook status %s", resp.Status)
		}
		last = err
		time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
	}
	return last
}

// SSE serves GET /events and GET /health on a loopback address.
type SSE struct {
	Addr string

	mu      sync.Mutex
	clients map[chan []byte]struct{}
	srv     *http.Server
}

func NewSSE(addr string) *SSE {
	return &SSE{Addr: addr, clients: map[chan []byte]struct{}{}}
}

// Start listens on Addr. Shutdown stops the server.
func (s *SSE) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/events", s.events)
	s.srv = &http.Server{
		Addr:              s.Addr,
		Handler:           mux,
		ErrorLog:          log.New(io.Discard, "", 0),
		ReadHeaderTimeout: 5 * time.Second,
	}
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	go func() {
		_ = s.srv.Serve(ln)
	}()
	return nil
}

// Shutdown stops the listener.
func (s *SSE) Shutdown(ctx context.Context) error {
	if s.srv == nil {
		return nil
	}
	return s.srv.Shutdown(ctx)
}

func (s *SSE) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ch := make(chan []byte, 16)
	s.mu.Lock()
	s.clients[ch] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.clients, ch)
		s.mu.Unlock()
	}()
	fmt.Fprintf(w, ": ok\n\n")
	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: vacancy\ndata: %s\n\n", msg)
			flusher.Flush()
		}
	}
}

func (s *SSE) Emit(h Hit) error {
	body, err := marshal(h)
	if err != nil {
		return err
	}
	body = bytes.TrimRight(body, "\n")
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.clients {
		select {
		case ch <- body:
		default:
		}
	}
	return nil
}

func marshal(h Hit) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(h); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Loopback reports whether addr binds only to the local machine.
func Loopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
