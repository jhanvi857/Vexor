package main

import (
	"fmt"
	"net/http"
	"sync"
)

func startBackend(port int, name string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "%s (port %d) OK\n", name, port)
	})

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	fmt.Printf("Starting %s on http://%s\n", name, addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Printf("Error on port %d: %v\n", port, err)
	}
}

func main() {
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		startBackend(3001, "Backend One")
	}()

	go func() {
		defer wg.Done()
		startBackend(3002, "Backend Two")
	}()

	go func() {
		defer wg.Done()
		startBackend(3003, "Backend Three")
	}()

	wg.Wait()
}
