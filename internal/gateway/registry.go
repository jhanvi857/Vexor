package gateway

import (
	"log"
	"strconv"
	"strings"

	"github.com/jhanvi857/vexor/internal/config"
	"github.com/jhanvi857/vexor/internal/load_balancer"
)

// balancers holds a Balancer per route path when a route exposes multiple targets.
var balancers map[string]load_balancer.Balancer

// InitBalancers builds balancers from config.Routes. If a route.Target contains
// multiple comma-separated URLs, a RoundRobin balancer is created and health
// checks are started for each instance.
func InitBalancers() {
	balancers = make(map[string]load_balancer.Balancer)
	for _, route := range config.Routes {
		var targets []string
		if len(route.Targets) > 0 {
			targets = route.Targets
		} else if strings.TrimSpace(route.Target) != "" {
			targets = []string{route.Target}
		}
		if len(targets) <= 1 {
			continue
		}
		instances := make([]*load_balancer.Instance, 0, len(targets))
		totalWeight := 0
		for _, t := range targets {
			u := strings.TrimSpace(t)
			weight := 1
			// allow optional weight suffix after '|' (e.g. url|2)
			if idx := strings.Index(u, "|"); idx != -1 {
				wstr := u[idx+1:]
				u = u[:idx]
				if parsed, err := strconv.Atoi(strings.TrimSpace(wstr)); err == nil && parsed > 0 {
					weight = parsed
				}
			}
			inst := &load_balancer.Instance{ID: u, URL: u, Weight: weight, Healthy: true}
			load_balancer.StartHealthCheck(inst)
			instances = append(instances, inst)
			totalWeight += weight
		}
		// choose strategy
		var b load_balancer.Balancer
		switch strings.ToLower(strings.TrimSpace(route.Strategy)) {
		case "least":
			b = load_balancer.NewLeastConnections(instances)
		case "weighted":
			b = load_balancer.NewWeightedRoundRobin(instances)
		default:
			b = load_balancer.NewRoundRobin(instances)
		}
		balancers[route.Path] = b
		log.Printf("initialized %T balancer for route %s with %d targets (totalWeight=%d)", b, route.Path, len(instances), totalWeight)
	}
}

// GetBalancer returns the Balancer for the given route path, or nil if none.
func GetBalancer(path string) load_balancer.Balancer {
	if balancers == nil {
		return nil
	}
	return balancers[path]
}
