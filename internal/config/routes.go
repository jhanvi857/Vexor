package config

// Route describes an upstream mapping.
// Use either Target (single backend) or Targets (multiple backends).
// Targets may optionally include a weight suffix using the form: "url|weight".
// Example: Targets: []string{"http://localhost:3001|1", "http://localhost:3002|2"}
type Route struct {
	Path    string   `yaml:"path" json:"path"`
	Target  string   `yaml:"target,omitempty" json:"target,omitempty"`   // legacy single target
	Targets []string `yaml:"targets,omitempty" json:"targets,omitempty"` // optional multiple targets
	// Strategy can be "round", "least", or "weighted". Defaults to "round".
	Strategy  string           `yaml:"strategy,omitempty" json:"strategy,omitempty"`
	RateLimit *RateLimitConfig `yaml:"rate_limit,omitempty" json:"rate_limit,omitempty"`
}

// Routes is populated from config.yaml at startup.
var Routes []Route

// TrustedProxies is populated from config.yaml at startup.
var TrustedProxies []string
