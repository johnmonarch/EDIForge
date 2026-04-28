package translate

import (
	"io"

	"github.com/johnmonarch/ediforge/internal/model"
)

type Input struct {
	Name        string
	Reader      io.Reader
	Size        int64
	ContentType string
}

type TranslateOptions struct {
	Standard        model.Standard `json:"standard,omitempty"`
	Mode            model.Mode     `json:"mode,omitempty"`
	SchemaPath      string         `json:"schema,omitempty"`
	SchemaID        string         `json:"schemaId,omitempty"`
	Pretty          bool           `json:"pretty,omitempty"`
	IncludeEnvelope bool           `json:"includeEnvelope,omitempty"`
	IncludeRaw      bool           `json:"includeRawSegments,omitempty"`
	IncludeOffsets  bool           `json:"includeOffsets,omitempty"`
	AllowPartial    bool           `json:"allowPartial,omitempty"`
}

type ValidateOptions struct {
	Standard   model.Standard `json:"standard,omitempty"`
	SchemaPath string         `json:"schema,omitempty"`
	SchemaID   string         `json:"schemaId,omitempty"`
	Level      string         `json:"level,omitempty"`
	Strict     bool           `json:"strict,omitempty"`
}

type DetectOptions struct {
	Standard model.Standard `json:"standard,omitempty"`
}
