package gateway

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jhanvi857/vexor/internal/breaker"
	"github.com/jhanvi857/vexor/internal/config"
	"github.com/jhanvi857/vexor/internal/load_balancer"
	"github.com/jhanvi857/vexor/internal/observability"
	strategy "github.com/jhanvi857/vexor/internal/ratelimit/strategy"
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

func TestGatewayHandlerEnforcesTrustedProxies(t *testing.T) {
	originalRoutes := config.Routes
	originalTrustedProxies := config.TrustedProxies
	defer func() {
		config.Routes = originalRoutes
		config.TrustedProxies = originalTrustedProxies
		InitBalancers()
	}()

	backend := startMockBackend(t, "backend")
	defer backend.Close()

	config.Routes = []config.Route{
		{Path: "/users", Target: backend.URL},
	}
	config.TrustedProxies = []string{"203.0.113.10"}

	InitBalancers()
	handler := GatewayHandler()

	// Request from untrusted IP should fail with 403
	reqUntrusted := httptest.NewRequest(http.MethodGet, "/users", nil)
	reqUntrusted.RemoteAddr = "198.51.100.5:12345"
	recUntrusted := httptest.NewRecorder()
	handler.ServeHTTP(recUntrusted, reqUntrusted)
	if recUntrusted.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for untrusted proxy, got %d", recUntrusted.Code)
	}

	// Request from trusted IP should succeed with 200
	reqTrusted := httptest.NewRequest(http.MethodGet, "/users", nil)
	reqTrusted.RemoteAddr = "203.0.113.10:12345"
	recTrusted := httptest.NewRecorder()
	handler.ServeHTTP(recTrusted, reqTrusted)
	if recTrusted.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for trusted proxy, got %d", recTrusted.Code)
	}
}

func TestGatewayHandlerLeastConnectionsTracksActiveConnections(t *testing.T) {
	originalRoutes := config.Routes
	originalTrustedProxies := config.TrustedProxies
	defer func() {
		config.Routes = originalRoutes
		config.TrustedProxies = originalTrustedProxies
		InitBalancers()
	}()

	// Create a slow backend that blocks until signaled, simulating active connection
	signalChan := make(chan struct{})
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		case "/users":
			<-signalChan // block until signaled
			_, _ = w.Write([]byte("slow-backend"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer backend.Close()

	config.Routes = []config.Route{
		{
			Path:     "/users",
			Targets:  []string{backend.URL},
			Strategy: "least",
		},
	}
	config.TrustedProxies = nil // trust all

	InitBalancers()
	handler := GatewayHandler()

	// Retrieve the load balancer and instances
	balancer := GetBalancer("/users")
	if balancer == nil {
		t.Fatal("expected balancer to be initialized")
	}
	lc, ok := balancer.(*load_balancer.LeastConnections)
	if !ok {
		t.Fatalf("expected LeastConnections balancer type, got %T", balancer)
	}

	// Start request in a goroutine
	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()
	doneChan := make(chan struct{})
	go func() {
		handler.ServeHTTP(rec, req)
		close(doneChan)
	}()

	// Wait a bit to ensure request reaches backend and blocks
	time.Sleep(50 * time.Millisecond)

	// Check active connections count
	inst, err := lc.NextInstance()
	if err != nil {
		t.Fatalf("NextInstance failed: %v", err)
	}
	active := atomic.LoadInt64(&inst.ActiveConnections)
	if active != 1 {
		t.Fatalf("expected ActiveConnections count to be 1, got %d", active)
	}

	// Signal backend to finish request
	close(signalChan)
	<-doneChan

	// Verify connection count drops back to 0
	activeAfter := atomic.LoadInt64(&inst.ActiveConnections)
	if activeAfter != 0 {
		t.Fatalf("expected ActiveConnections count to return to 0, got %d", activeAfter)
	}
}

func TestGatewayMetricsEndpointExposesSnapshots(t *testing.T) {
	originalRoutes := config.Routes
	originalTrustedProxies := config.TrustedProxies
	defer func() {
		config.Routes = originalRoutes
		config.TrustedProxies = originalTrustedProxies
		InitBalancers()
	}()

	backend := startMockBackend(t, "backend")
	defer backend.Close()

	config.Routes = []config.Route{
		{
			Path:   "/users",
			Target: backend.URL,
			RateLimit: &config.RateLimitConfig{
				Strategy: "fixed_window",
				Limit:    10,
			},
		},
	}
	config.TrustedProxies = nil // trust all

	InitBalancers()

	metricsRegistry := observability.NewRegistry()
	metricsRegistry.Register("breakers", func() interface{} {
		snapshots := make(map[string]interface{})
		for path, cb := range breakers {
			if cb != nil {
				snapshots[path] = breaker.Snapshot(cb)
			}
		}
		return snapshots
	})
	metricsRegistry.Register("rate_limiters", func() interface{} {
		snapshots := make(map[string]interface{})
		for path, rl := range rateLimiters {
			if rl != nil {
				if snap, ok := rl.(*strategy.TokenBucket); ok {
					snapshots[path] = snap.Snapshot()
				} else if snap, ok := rl.(*strategy.FixedWindow); ok {
					snapshots[path] = snap.Snapshot()
				} else if snap, ok := rl.(*strategy.SlidingCounter); ok {
					snapshots[path] = snap.Snapshot()
				} else if snap, ok := rl.(*strategy.SlidingLog); ok {
					snapshots[path] = snap.Snapshot()
				}
			}
		}
		return snapshots
	})

	handler := GatewayHandler()
	performGatewayRequest(t, handler, "/users")

	snap := metricsRegistry.Snapshot()

	breakersMap, ok := snap["breakers"].(map[string]interface{})
	if !ok {
		t.Fatal("expected breakers map in snapshot")
	}
	if _, exists := breakersMap["/users"]; !exists {
		t.Fatal("expected breaker metrics for /users")
	}

	limitersMap, ok := snap["rate_limiters"].(map[string]interface{})
	if !ok {
		t.Fatal("expected rate_limiters map in snapshot")
	}
	if _, exists := limitersMap["/users"]; !exists {
		t.Fatal("expected rate limiter metrics for /users")
	}
}

func TestGatewayFixedWindowHonorsCustomWindowSeconds(t *testing.T) {
	originalRoutes := config.Routes
	originalTrustedProxies := config.TrustedProxies
	defer func() {
		config.Routes = originalRoutes
		config.TrustedProxies = originalTrustedProxies
		InitBalancers()
	}()

	backend := startMockBackend(t, "backend")
	defer backend.Close()

	config.Routes = []config.Route{
		{
			Path:   "/users",
			Target: backend.URL,
			RateLimit: &config.RateLimitConfig{
				Strategy:   "fixed_window",
				Limit:      1,
				WindowSecs: 2,
			},
		},
	}
	config.TrustedProxies = nil // trust all

	InitBalancers()
	handler := GatewayHandler()

	// First request succeeds
	performGatewayRequest(t, handler, "/users")

	// Second request within the 2-second window fails with 429
	req2 := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request within window to be rate limited, got %d", rec2.Code)
	}

	// Sleep 2.1 seconds to cross the window boundary
	time.Sleep(2100 * time.Millisecond)

	// Third request should succeed
	req3 := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("expected request after window boundary to succeed, got %d", rec3.Code)
	}
}
