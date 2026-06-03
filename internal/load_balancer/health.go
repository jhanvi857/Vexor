package load_balancer

import (
	"io"
	"net/http"
	"strings"
	"time"
)

func StartHealthCheck(instance *Instance) {
	client := http.Client{
		Timeout: 3 * time.Second,
	}
	instance.SetHealthy(checkInstanceHealth(&client, instance.URL))
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		check := func() {
			instance.SetHealthy(checkInstanceHealth(&client, instance.URL))
		}
		for range ticker.C {
			check()
		}
	}()
}

func checkInstanceHealth(client *http.Client, baseURL string) bool {
	healthURL := strings.TrimRight(baseURL, "/") + "/health"
	for _, method := range []string{http.MethodHead, http.MethodGet} {
		req, err := http.NewRequest(method, healthURL, nil)
		if err != nil {
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			return true
		}
	}
	return false
}
