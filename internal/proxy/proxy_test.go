package proxy

import (
	"net/http"
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
