package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewProxyConfiguresTimeoutTransport(t *testing.T) {
	p, err := NewProxy("http://example.com")
	if err != nil {
		t.Fatalf("new proxy: %v", err)
	}

	transport, ok := p.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected http.Transport, got %T", p.Transport)
	}
	if transport.ResponseHeaderTimeout != 10*time.Second {
		t.Fatalf("expected response header timeout of 10s, got %v", transport.ResponseHeaderTimeout)
	}
	if transport.DialContext == nil {
		t.Fatal("expected dial context to be configured")
	}
}

func TestNewProxySetsClientForwardingHeaders(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Forwarded-For"); got != "203.0.113.10" {
			t.Fatalf("expected X-Forwarded-For to contain client IP, got %q", got)
		}
		if got := r.Header.Get("X-Real-IP"); got != "203.0.113.10" {
			t.Fatalf("expected X-Real-IP to contain client IP, got %q", got)
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer backend.Close()

	p, err := NewProxy(backend.URL)
	if err != nil {
		t.Fatalf("new proxy: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://gateway.local/users", nil)
	req.RemoteAddr = "203.0.113.10:54321"
	req.Header.Set("X-Forwarded-For", "198.51.100.7")
	req.Header.Set("X-Real-IP", "198.51.100.8")

	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected proxy request to succeed, got %d with body %q", rec.Code, rec.Body.String())
	}
}
