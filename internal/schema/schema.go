package schema

import "github.com/johnmonarch/ediforge/internal/model"

type Schema struct {
	ID           string                 `json:"id"`
	Standard     model.Standard         `json:"standard"`
	Transaction  string                 `json:"transaction,omitempty"`
	Message      string                 `json:"message,omitempty"`
	Version      string                 `json:"version,omitempty"`
	Name         string                 `json:"name,omitempty"`
	License      string                 `json:"license,omitempty"`
	Source       string                 `json:"source,omitempty"`
	DocumentType string                 `json:"documentType,omitempty"`
	Segments     []SegmentRule          `json:"segments,omitempty"`
	Output       Output                 `json:"output"`
	Maps         map[string]string      `json:"maps"`
	Mapping      map[string]MappingRule `json:"mapping,omitempty"`
}

type Output struct {
	DocumentType string         `json:"documentType,omitempty"`
	Type         string         `json:"type,omitempty"`
	Required     []string       `json:"required,omitempty"`
	Fields       map[string]any `json:"fields,omitempty"`
}

type SegmentRule struct {
	Tag      string            `json:"tag"`
	Purpose  string            `json:"purpose,omitempty"`
	Required bool              `json:"required,omitempty"`
	Max      int               `json:"max,omitempty"`
	Loop     string            `json:"loop,omitempty"`
	Maps     map[string]string `json:"maps,omitempty"`
}

type MappingRule struct {
	Literal    string   `json:"literal,omitempty"`
	Path       string   `json:"path,omitempty"`
	Transforms []string `json:"transforms,omitempty"`
}

type Summary struct {
	ID          string         `json:"id"`
	Standard    model.Standard `json:"standard"`
	Transaction string         `json:"transaction,omitempty"`
	Message     string         `json:"message,omitempty"`
	Name        string         `json:"name,omitempty"`
	Path        string         `json:"path,omitempty"`
}
