package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsRoutesFromYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte("routes:\n  - path: /users\n    target: http://example.com\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	routes, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(routes) != 1 || routes[0].Path != "/users" || routes[0].Target != "http://example.com" {
		t.Fatalf("unexpected routes: %#v", routes)
	}
}
