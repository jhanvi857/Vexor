package ratelimit

import (
	"net/http"

	strategy "github.com/jhanvi857/vexor/internal/ratelimit/strategy"
)

func Middleware(limiter strategy.Strategy, keyExtractor func(r *http.Request) string) func(http.Handler) http.Handler {
	if keyExtractor == nil {
		keyExtractor = func(r *http.Request) string {
			return r.RemoteAddr
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = keyExtractor(r)
			if limiter != nil && !limiter.Allow() {
				w.Header().Set("Retry-After", "1")
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
