package routing

import (
	"testing"

	"github.com/jhanvi857/vexor/internal/config"
)

func TestMatchRouteExactPrefix(t *testing.T) {
	original := config.Routes
	defer func() { config.Routes = original }()

	config.Routes = []config.Route{{Path: "/users", Target: "http://users"}}

	route := MatchRoute("/users")
	if route.Path != "/users" {
		t.Fatalf("expected exact prefix match, got %#v", route)
	}
}

func TestMatchRouteLongestPrefixWins(t *testing.T) {
	original := config.Routes
	defer func() { config.Routes = original }()

	config.Routes = []config.Route{
		{Path: "/api", Target: "http://api"},
		{Path: "/api/v1", Target: "http://api-v1"},
	}

	route := MatchRoute("/api/v1/users")
	if route.Target != "http://api-v1" {
		t.Fatalf("expected longest prefix to win, got %#v", route)
	}
}

func TestMatchRouteNoMatchReturnsEmptyString(t *testing.T) {
	original := config.Routes
	defer func() { config.Routes = original }()

	config.Routes = []config.Route{{Path: "/users", Target: "http://users"}}

	route := MatchRoute("/orders")
	if route.Path != "" || route.Target != "" {
		t.Fatalf("expected zero-value route for no match, got %#v", route)
	}
}
