package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/openedi/ediforge/internal/schema"
)

func NewSchemaRegistry(cfg Config) *schema.Registry {
	return AugmentSchemaRegistry(schema.NewRegistry(), cfg)
}

func AugmentSchemaRegistry(registry *schema.Registry, cfg Config) *schema.Registry {
	if registry == nil {
		registry = schema.NewRegistry()
	}
	roots := make([]string, 0, len(cfg.Schemas.Paths)+len(registry.Roots))
	seen := map[string]bool{}
	for _, root := range cfg.Schemas.Paths {
		addRoot(&roots, seen, expandHome(root))
	}
	for _, root := range registry.Roots {
		addRoot(&roots, seen, root)
	}
	registry.Roots = roots
	return registry
}

func addRoot(roots *[]string, seen map[string]bool, root string) {
	root = strings.TrimSpace(root)
	if root == "" || seen[root] {
		return
	}
	seen[root] = true
	*roots = append(*roots, root)
}

func expandHome(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return path
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
