package gateway

import (
	"log"
	"net/http"
)

func Start() {
	http.HandleFunc("/", GatewayHandler)
	log.Println("Gateway started at port : 8080")
	http.ListenAndServe(":8080", nil)
}
