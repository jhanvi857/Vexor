package gateway

import (
	"log"
	"net/http"
	"time"

	"github.com/jhanvi857/vexor/internal/middleware"
	"github.com/jhanvi857/vexor/internal/observability"
	"github.com/jhanvi857/vexor/internal/ratelimit"
	strategy "github.com/jhanvi857/vexor/internal/ratelimit/strategy"
)

func Start() {
	rateLimiter := &strategy.TokenBucket{
		Capacity:       100,
		Tokens:         100,
		RefillRate:     10,
		LastRefillTime: time.Now(),
	}
	metricsRegistry := observability.NewRegistry()
	metricsRegistry.Register("ratelimit_token_bucket", func() interface{} {
		return rateLimiter.Snapshot()
	})

	handler := GatewayHandler()
	finalHandler := middleware.Chain(handler, middleware.Recovery, middleware.RequestID, ratelimit.Middleware(rateLimiter, nil), middleware.Logger)
	http.Handle("/metrics", observability.JSONHandler(metricsRegistry))
	http.Handle("/", finalHandler)
	log.Println("API gateway running on port : 8080")
	http.ListenAndServe(":8080", nil)
}
