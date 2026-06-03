package gateway

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jhanvi857/vexor/internal/config"
	"github.com/jhanvi857/vexor/internal/middleware"
	"github.com/jhanvi857/vexor/internal/observability"
)

func Start() {
	if routes, err := config.LoadDefault(); err != nil {
		log.Fatalf("load config: %v", err)
	} else {
		config.Routes = routes
	}

	metricsRegistry := observability.NewRegistry()
	metricsRegistry.Register("routes", func() interface{} { return config.Routes })

	// initialize balancers, breakers, and route-local rate limiters from config.
	InitRoutePolicies()

	mux := http.NewServeMux()
	mux.Handle("/metrics", observability.JSONHandler(metricsRegistry))
	mux.Handle("/", middleware.Chain(GatewayHandler(), middleware.Recovery, middleware.RequestID, middleware.Logger))

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		log.Println("API gateway running on port : 8080")
		serverErr <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}
}
