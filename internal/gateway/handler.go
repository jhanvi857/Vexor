package gateway

import (
	"net/http"
	"strconv"
	"sync/atomic"

	"github.com/jhanvi857/vexor/internal/load_balancer"
	"github.com/jhanvi857/vexor/internal/proxy"
	"github.com/jhanvi857/vexor/internal/routing"
)

func GatewayHandler() http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if !proxy.IsTrustedProxy(r.RemoteAddr) {
			http.Error(w, "Forbidden: Untrusted socket peer", http.StatusForbidden)
			return
		}

		route := routing.MatchRoute(r.URL.Path)

		if route.Path == "" {
			http.NotFound(w, r)
			return
		}

		if routeLimiter := GetRateLimiter(route.Path); routeLimiter != nil && !routeLimiter.Allow() {
			w.Header().Set("Retry-After", "1")
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		routeBreaker := GetBreaker(route.Path)
		if routeBreaker != nil && !routeBreaker.AllowRequest() {
			retry := int(routeBreaker.ResetTimeout.Seconds())
			if retry < 1 {
				retry = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(retry))
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}

		// If a balancer exists for this route (multiple targets configured),
		// select an instance from the balancer and proxy to that instance.
		target := route.Target
		var selectedInstance *load_balancer.Instance
		if b := GetBalancer(route.Path); b != nil {
			inst, err := b.NextInstance()
			if err != nil {
				if routeBreaker != nil {
					routeBreaker.RecordFailure()
				}
				http.Error(w, "Bad Gateway", http.StatusBadGateway)
				return
			}
			target = inst.URL
			selectedInstance = inst
		}

		if selectedInstance != nil {
			atomic.AddInt64(&selectedInstance.ActiveConnections, 1)
			defer atomic.AddInt64(&selectedInstance.ActiveConnections, -1)
		}

		proxyInstance, err := proxy.NewProxy(target)

		if err != nil {
			if routeBreaker != nil {
				routeBreaker.RecordFailure()
			}
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		if routeBreaker != nil {
			proxyInstance.ModifyResponse = func(resp *http.Response) error {
				if resp.StatusCode >= http.StatusInternalServerError {
					routeBreaker.RecordFailure()
				} else {
					routeBreaker.RecordSuccess()
				}
				return nil
			}
			proxyInstance.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
				routeBreaker.RecordFailure()
				http.Error(w, "Bad Gateway", http.StatusBadGateway)
			}
		}

		proxyInstance.ServeHTTP(w, r)
	})
}
