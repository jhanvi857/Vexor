package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config describes the runtime gateway configuration file.
type Config struct {
	Routes         []Route  `yaml:"routes" json:"routes"`
	TrustedProxies []string `yaml:"trusted_proxies" json:"trusted_proxies"`
}

// RateLimitConfig configures a per-route limiter.
type RateLimitConfig struct {
	Strategy   string  `yaml:"strategy" json:"strategy"`
	Capacity   int     `yaml:"capacity,omitempty" json:"capacity,omitempty"`
	Tokens     float64 `yaml:"tokens,omitempty" json:"tokens,omitempty"`
	RefillRate float64 `yaml:"refill_rate,omitempty" json:"refill_rate,omitempty"`
	Limit      int     `yaml:"limit,omitempty" json:"limit,omitempty"`
	WindowSecs int     `yaml:"window_seconds,omitempty" json:"window_seconds,omitempty"`
	NumBuckets int     `yaml:"num_buckets,omitempty" json:"num_buckets,omitempty"`
}

const defaultConfigFile = "config.yaml"

// Load reads routes from a YAML or JSON config file.
func Load(path string) ([]Route, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	Routes = cfg.Routes
	TrustedProxies = cfg.TrustedProxies
	return cfg.Routes, nil
}

// LoadDefault loads configuration from config.yaml in the current working directory,
// or from the VEXOR_CONFIG environment variable when set.
func LoadDefault() ([]Route, error) {
	path := strings.TrimSpace(os.Getenv("VEXOR_CONFIG"))
	if path == "" {
		path = defaultConfigFile
	}
	if !filepath.IsAbs(path) {
		if wd, err := os.Getwd(); err == nil {
			path = filepath.Join(wd, path)
		}
	}
	return Load(path)
}
