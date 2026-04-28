package schema

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Registry struct {
	Roots []string
}

func NewRegistry() *Registry {
	return &Registry{Roots: []string{"schemas/examples"}}
}

func (r *Registry) Resolve(id, path string) (*Schema, error) {
	if path != "" {
		return LoadFile(path)
	}
	if id == "" {
		return nil, errors.New("semantic mode requires --schema or --schema-id")
	}
	for _, root := range r.Roots {
		matches, _ := filepath.Glob(filepath.Join(root, id+".*"))
		for _, match := range matches {
			schema, err := LoadFile(match)
			if err == nil && schema.ID == id {
				return schema, nil
			}
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !schemaExt(entry.Name()) {
				continue
			}
			schema, err := LoadFile(filepath.Join(root, entry.Name()))
			if err == nil && schema.ID == id {
				return schema, nil
			}
		}
	}
	return nil, fmt.Errorf("schema %q not found", id)
}

func (r *Registry) List() ([]Summary, error) {
	var summaries []Summary
	for _, root := range r.Roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !schemaExt(entry.Name()) {
				continue
			}
			path := filepath.Join(root, entry.Name())
			loaded, err := LoadFile(path)
			if err != nil {
				continue
			}
			summaries = append(summaries, Summary{
				ID:          loaded.ID,
				Standard:    loaded.Standard,
				Transaction: loaded.Transaction,
				Message:     loaded.Message,
				Name:        loaded.Name,
				Path:        path,
			})
		}
	}
	return summaries, nil
}

func schemaExt(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".json" || ext == ".yml" || ext == ".yaml"
}

func LoadFile(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	var loaded *Schema
	switch ext {
	case ".json":
		var s Schema
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, err
		}
		loaded = &s
	case ".yml", ".yaml":
		s, err := parseSimpleYAML(string(data))
		if err != nil {
			return nil, err
		}
		loaded = s
	default:
		return nil, fmt.Errorf("unsupported schema extension %q", ext)
	}
	loaded.normalize()
	if err := Validate(loaded); err != nil {
		return nil, err
	}
	return loaded, nil
}

func (s *Schema) normalize() {
	if s.Output.DocumentType == "" {
		s.Output.DocumentType = s.DocumentType
	}
	if s.Maps == nil {
		s.Maps = map[string]string{}
	}
	for target, rule := range s.Mapping {
		if rule.Path == "" {
			continue
		}
		if strings.Contains(target, "[]") || strings.Contains(rule.Path, "[]") || strings.Contains(rule.Path, ">") {
			continue
		}
		expression := rule.Path
		for _, transform := range rule.Transforms {
			expression += " | " + normalizeTransform(transform)
		}
		s.Maps[target] = expression
	}
}

func normalizeTransform(transform string) string {
	transform = strings.TrimSpace(transform)
	if strings.HasPrefix(transform, "date:") {
		return "date('" + strings.TrimPrefix(transform, "date:") + "')"
	}
	return transform
}
