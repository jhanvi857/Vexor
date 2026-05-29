package gateway

import (
	"log"
	"net/http"

	"github.com/jhanvi857/vexor/internal/proxy"
	"github.com/jhanvi857/vexor/internal/routing"
)

func GatewayHandler(w http.ResponseWriter, r *http.Request) {
	route := routing.MatchRoute(r.URL.Path)
	if route == nil {
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}

	proxy, err := proxy.NewProxy(route.Target)
	if err != nil {
		log.Printf("Error creating proxy for route %s: %v", route.Path, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	proxy.ServeHTTP(w, r)
}
