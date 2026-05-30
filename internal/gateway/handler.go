package gateway

import (
	"net/http"

	"github.com/jhanvi857/vexor/internal/proxy"
	"github.com/jhanvi857/vexor/internal/routing"
)

func GatewayHandler() http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		route := routing.MatchRoute(r.URL.Path)

		if route.Path == "" {
			http.NotFound(w, r)
			return
		}

		proxyInstance, err := proxy.NewProxy(route.Target)

		if err != nil {
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
			return
		}

		proxyInstance.ServeHTTP(w, r)
	})
}
