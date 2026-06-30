package gateway

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jhanvi857/vexor/internal/breaker"
	"github.com/jhanvi857/vexor/internal/config"
	"github.com/jhanvi857/vexor/internal/load_balancer"
	strategy "github.com/jhanvi857/vexor/internal/ratelimit/strategy"
)

// balancers holds a Balancer per route path when a route exposes multiple targets.
var balancers map[string]load_balancer.Balancer

// breakers holds a circuit breaker per route path.
var breakers map[string]*breaker.CircuitBreaker

// rateLimiters holds a rate limiter per route path.
var rateLimiters map[string]strategy.Strategy

const (
	defaultBreakerFailureThreshold = 3
	defaultBreakerResetTimeout     = 30 * time.Second
	defaultTokenBucketCapacity     = 100
	defaultTokenBucketRefillRate   = 10
)

// InitBalancers builds balancers from config.Routes. If a route.Target contains
// multiple comma-separated URLs, a RoundRobin balancer is created and health
// checks are started for each instance.
func InitBalancers() {
	InitRoutePolicies()
}

// InitRoutePolicies builds balancers, breakers, and per-route rate limiters.
func InitRoutePolicies() {
	balancers = make(map[string]load_balancer.Balancer)
	breakers = make(map[string]*breaker.CircuitBreaker)
	rateLimiters = make(map[string]strategy.Strategy)
	for _, route := range config.Routes {
		var targets []string
		if len(route.Targets) > 0 {
			targets = route.Targets
		} else if strings.TrimSpace(route.Target) != "" {
			targets = []string{route.Target}
		}
		if len(targets) == 0 {
			continue
		}
		instances := make([]*load_balancer.Instance, 0, len(targets))
		totalWeight := 0
		for _, t := range targets {
			u := strings.TrimSpace(t)
			weight := 1
			// allow optional weight suffix after '|' (e.g. url|2)
			if idx := strings.Index(u, "|"); idx != -1 {
				wstr := u[idx+1:]
				u = u[:idx]
				if parsed, err := strconv.Atoi(strings.TrimSpace(wstr)); err == nil && parsed > 0 {
					weight = parsed
				}
			}
			inst := &load_balancer.Instance{ID: u, URL: u, Weight: weight, Healthy: true}
			load_balancer.StartHealthCheck(inst)
			instances = append(instances, inst)
			totalWeight += weight
		}
		// choose strategy
		var b load_balancer.Balancer
		switch strings.ToLower(strings.TrimSpace(route.Strategy)) {
		case "least":
			b = load_balancer.NewLeastConnections(instances)
		case "weighted":
			b = load_balancer.NewWeightedRoundRobin(instances)
		default:
			b = load_balancer.NewRoundRobin(instances)
		}
		balancers[route.Path] = b
		breakers[route.Path] = breaker.NewCircuitBreaker(defaultBreakerFailureThreshold, defaultBreakerResetTimeout)
		rateLimiters[route.Path] = buildRateLimiter(route)
		log.Printf("initialized %T balancer for route %s with %d targets (totalWeight=%d)", b, route.Path, len(instances), totalWeight)
	}
}

// GetBalancer returns the Balancer for the given route path, or nil if none.
func GetBalancer(path string) load_balancer.Balancer {
	if balancers == nil {
		return nil
	}
	return balancers[path]
}

// GetBreaker returns the circuit breaker for the given route path, or nil if none.
func GetBreaker(path string) *breaker.CircuitBreaker {
	if breakers == nil {
		return nil
	}
	return breakers[path]
}

// GetRateLimiter returns the limiter for the given route path, or nil if none.
func GetRateLimiter(path string) strategy.Strategy {
	if rateLimiters == nil {
		return nil
	}
	return rateLimiters[path]
}

func buildRateLimiter(route config.Route) strategy.Strategy {
	conf := route.RateLimit
	if conf == nil {
		return &strategy.TokenBucket{
			Capacity:       defaultTokenBucketCapacity,
			Tokens:         float64(defaultTokenBucketCapacity),
			RefillRate:     defaultTokenBucketRefillRate,
			LastRefillTime: time.Now(),
		}
	}

	switch strings.ToLower(strings.TrimSpace(conf.Strategy)) {
	case "fixed_window":
		limit := conf.Limit
		if limit <= 0 {
			limit = 100
		}
		window := time.Duration(conf.WindowSecs) * time.Second
		if window <= 0 {
			window = time.Minute
		}
		return &strategy.FixedWindow{Limit: limit, Window: window, WindowStart: time.Now()}
	case "sliding_counter":
		limit := conf.Limit
		if limit <= 0 {
			limit = 100
		}
		window := time.Duration(conf.WindowSecs) * time.Second
		if window <= 0 {
			window = time.Minute
		}
		buckets := conf.NumBuckets
		if buckets <= 0 {
			buckets = 10
		}
		return &strategy.SlidingCounter{Limit: limit, Window: window, NumBuckets: buckets}
	case "sliding_log":
		limit := conf.Limit
		if limit <= 0 {
			limit = 100
		}
		window := time.Duration(conf.WindowSecs) * time.Second
		if window <= 0 {
			window = time.Minute
		}
		return &strategy.SlidingLog{Limit: limit, Window: window}
	default:
		capacity := conf.Capacity
		if capacity <= 0 {
			capacity = defaultTokenBucketCapacity
		}
		refillRate := conf.RefillRate
		if refillRate <= 0 {
			refillRate = defaultTokenBucketRefillRate
		}
		tokens := conf.Tokens
		if tokens <= 0 {
			tokens = float64(capacity)
		}
		return &strategy.TokenBucket{
			Capacity:       capacity,
			Tokens:         tokens,
			RefillRate:     refillRate,
			LastRefillTime: time.Now(),
		}
	}
}
