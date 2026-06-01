package load_balancer

import (
	"net/http"
	"time"
)

func StartHealthCheck(instance *Instance) {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			client := http.Client{
				Timeout: 3 * time.Second,
			}
			resp, err := client.Get(instance.URL + "/health")
			if err != nil {
				instance.Healthy = false
				continue
			}
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				instance.Healthy = true
			} else {
				instance.Healthy = false
			}
		}
	}()
}
