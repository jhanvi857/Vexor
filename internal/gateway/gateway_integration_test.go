package gateway

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jhanvi857/vexor/internal/config"
)

func TestGatewayHandlerRoutesToWeightedBackends(t *testing.T) {
	originalRoutes := config.Routes
	defer func() {
		config.Routes = originalRoutes
		InitBalancers()
	}()

	backendOne := startMockBackend(t, "backend-one")
	backendTwo := startMockBackend(t, "backend-two")

	config.Routes = []config.Route{
		{
			Path:     "/users",
			Targets:  []string{backendOne.URL + "|1", backendTwo.URL + "|2"},
			Strategy: "weighted",
		},
	}

	InitBalancers()
	handler := GatewayHandler()

	first := performGatewayRequest(t, handler, "/users")
	second := performGatewayRequest(t, handler, "/users")

	if first == second {
		t.Fatalf("expected weighted round robin to reach different backends across requests, got %q twice", first)
	}

	if first != "backend-one" && first != "backend-two" {
		t.Fatalf("unexpected first backend response %q", first)
	}
	if second != "backend-one" && second != "backend-two" {
		t.Fatalf("unexpected second backend response %q", second)
	}
}

func TestGatewayHandlerKeepsRouteCircuitBreakersIsolated(t *testing.T) {
	originalRoutes := config.Routes
	defer func() {
		config.Routes = originalRoutes
		InitBalancers()
	}()

	healthyBackend := startMockBackend(t, "healthy-backend")
	failingBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		case "/orders":
			http.Error(w, "boom", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer failingBackend.Close()

	config.Routes = []config.Route{
		{Path: "/users", Target: healthyBackend.URL},
		{Path: "/orders", Target: failingBackend.URL},
	}

	InitBalancers()
	handler := GatewayHandler()

	for i := 0; i < 3; i++ {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/orders", nil)
		handler.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("expected /orders to fail with 500, got %d", recorder.Code)
		}
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected /orders breaker to open after failures, got %d", recorder.Code)
	}

	if got := performGatewayRequest(t, handler, "/users"); got != "healthy-backend" {
		t.Fatalf("expected /users to remain healthy, got %q", got)
	}
}

func TestGatewayHandlerSkipsUnhealthyBackends(t *testing.T) {
	originalRoutes := config.Routes
	defer func() {
		config.Routes = originalRoutes
		InitBalancers()
	}()

	healthyBackend := startMockBackend(t, "healthy-backend")
	unhealthyBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			_, _ = fmt.Fprint(w, "unhealthy-backend")
		}
	}))
	defer unhealthyBackend.Close()

	config.Routes = []config.Route{
		{
			Path:     "/users",
			Targets:  []string{unhealthyBackend.URL, healthyBackend.URL},
			Strategy: "round",
		},
	}

	InitBalancers()
	handler := GatewayHandler()

	if got := performGatewayRequest(t, handler, "/users"); got != "healthy-backend" {
		t.Fatalf("expected unhealthy backend to be skipped, got %q", got)
	}
}

func TestGatewayHandlerAppliesPerRouteRateLimiters(t *testing.T) {
	originalRoutes := config.Routes
	defer func() {
		config.Routes = originalRoutes
		InitBalancers()
	}()

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		default:
			_, _ = fmt.Fprint(w, "backend")
		}
	}))
	defer backend.Close()
	config.Routes = []config.Route{
		{
			Path:   "/users",
			Target: backend.URL,
			RateLimit: &config.RateLimitConfig{
				Strategy: "fixed_window",
				Limit:    1,
			},
		},
		{
			Path:   "/orders",
			Target: backend.URL,
			RateLimit: &config.RateLimitConfig{
				Strategy: "fixed_window",
				Limit:    1,
			},
		},
	}

	InitBalancers()
	handler := GatewayHandler()

	if got := performGatewayRequest(t, handler, "/users"); got != "backend" {
		t.Fatalf("expected first /users request to pass, got %q", got)
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected /users to be limited independently, got %d", recorder.Code)
	}

	if got := performGatewayRequest(t, handler, "/orders"); got != "backend" {
		t.Fatalf("expected /orders limiter to remain independent, got %q", got)
	}
}

func TestGatewayHandlerReturnsNotFoundForUnknownRoute(t *testing.T) {
	originalRoutes := config.Routes
	defer func() {
		config.Routes = originalRoutes
		InitBalancers()
	}()

	config.Routes = []config.Route{{Path: "/users", Target: "http://example.invalid"}}
	InitBalancers()

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	GatewayHandler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown route, got %d", recorder.Code)
	}
}

func startMockBackend(t *testing.T, name string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		case "/users":
			_, _ = fmt.Fprint(w, name)
		default:
			http.NotFound(w, r)
		}
	}))
}

func performGatewayRequest(t *testing.T, handler http.Handler, path string) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 from gateway for %s, got %d with body %q", path, recorder.Code, recorder.Body.String())
	}

	return strings.TrimSpace(recorder.Body.String())
}
