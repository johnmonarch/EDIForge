package translate

import (
	"github.com/johnmonarch/ediforge/internal/detect"
	"github.com/johnmonarch/ediforge/internal/model"
)

type TranslateResult struct {
	OK           bool               `json:"ok"`
	Standard     model.Standard     `json:"standard"`
	DocumentType string             `json:"documentType,omitempty"`
	Mode         model.Mode         `json:"mode"`
	Result       any                `json:"result,omitempty"`
	Warnings     []model.EDIWarning `json:"warnings,omitempty"`
	Errors       []model.EDIError   `json:"errors,omitempty"`
	Metadata     model.Metadata     `json:"metadata,omitempty"`
}

type ValidateResult struct {
	OK       bool               `json:"ok"`
	Standard model.Standard     `json:"standard"`
	Warnings []model.EDIWarning `json:"warnings,omitempty"`
	Errors   []model.EDIError   `json:"errors,omitempty"`
	Metadata model.Metadata     `json:"metadata,omitempty"`
}

type DetectResult = detect.Result
