package config

// Route describes an upstream mapping.
// Use either Target (single backend) or Targets (multiple backends).
// Targets may optionally include a weight suffix using the form: "url|weight".
// Example: Targets: []string{"http://localhost:3001|1", "http://localhost:3002|2"}
type Route struct {
	Path    string
	Target  string   // legacy single target
	Targets []string // optional multiple targets
	// Strategy can be "round", "least", or "weighted". Defaults to "round".
	Strategy string
}

var Routes = []Route{
	{
		Path:     "/users",
		Targets:  []string{"http://localhost:3001|1", "http://localhost:3003|2"},
		Strategy: "weighted",
	},
	{
		Path:   "/orders",
		Target: "http://localhost:3002",
	},
}
