package breaker

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type statusRecorder struct {
	http.ResponseWriter
	Status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.Status = code
	r.ResponseWriter.WriteHeader(code)
}

// Middleware returns an HTTP middleware that enforces the circuit breaker.
// When the breaker is OPEN, it returns 503 and a Retry-After header.
// After the wrapped handler runs, it records success for <500 responses
// and failure for >=500 responses or panics.
func Middleware(cb *CircuitBreaker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cb.AllowRequest() {
				retry := int(cb.ResetTimeout.Seconds())
				if retry < 1 {
					retry = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(retry))
				http.Error(w, "service unavailable", http.StatusServiceUnavailable)
				return
			}

			rec := &statusRecorder{ResponseWriter: w, Status: http.StatusOK}

			defer func() {
				if rec.Status >= 500 {
					cb.RecordFailure()
				} else {
					cb.RecordSuccess()
				}
			}()

			defer func() {
				if rec := recover(); rec != nil {
					cb.RecordFailure()
					panic(rec)
				}
			}()

			next.ServeHTTP(rec, r)
		})
	}
}

func MetricsHandler(cb *CircuitBreaker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := Snapshot(cb)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(m)
	}
}
