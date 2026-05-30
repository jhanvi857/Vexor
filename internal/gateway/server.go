package gateway

import (
	"log"
	"net/http"

	"github.com/jhanvi857/vexor/internal/middleware"
)

func Start() {
	handler := GatewayHandler()
	finalHandler := middleware.Chain(handler, middleware.Recovery, middleware.RequestID, middleware.Logger)
	http.Handle("/", finalHandler)
	log.Println("API gateway running on port : 8080")
	http.ListenAndServe(":8080", nil)
}
