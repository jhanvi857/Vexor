package routing

import (
	"strings"

	"github.com/jhanvi857/vexor/internal/config"
)

func MatchRoute(path string) *config.Route {
	for i := range config.Routes {
		route := &config.Routes[i]
		if strings.HasPrefix(path, route.Path) {
			return route
		}
	}
	return nil
}
