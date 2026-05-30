package routing

import (
	"strings"

	"github.com/jhanvi857/vexor/internal/config"
)

func MatchRoute(path string) config.Route {
	matched := config.Route{}
	matchedLength := -1
	for i := range config.Routes {
		route := config.Routes[i]
		if strings.HasPrefix(path, route.Path) && len(route.Path) > matchedLength {
			matched = route
			matchedLength = len(route.Path)
		}
	}
	return matched
}
