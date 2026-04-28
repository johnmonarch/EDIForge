package translate

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/johnmonarch/ediforge/internal/detect"
	"github.com/johnmonarch/ediforge/internal/jsonout"
	"github.com/johnmonarch/ediforge/internal/mapping"
	"github.com/johnmonarch/ediforge/internal/model"
	"github.com/johnmonarch/ediforge/internal/parse/edifact"
	"github.com/johnmonarch/ediforge/internal/parse/x12"
	"github.com/johnmonarch/ediforge/internal/schema"
	schemavalidate "github.com/johnmonarch/ediforge/internal/validate"
)

type Service struct {
	Schemas *schema.Registry
}

func NewService() *Service {
	return &Service{Schemas: schema.NewRegistry()}
}

func (s *Service) Detect(ctx context.Context, input Input, opts DetectOptions) (*DetectResult, error) {
	data, err := readAll(ctx, input.Reader)
	if err != nil {
		return nil, err
	}
	result, err := detect.Detect(data, opts.Standard)
	if err != nil {
		return &result, err
	}
	return &result, nil
}

func (s *Service) Translate(ctx context.Context, input Input, opts TranslateOptions) (*TranslateResult, error) {
	if opts.Standard == "" {
		opts.Standard = model.StandardAuto
	}
	if opts.Mode == "" {
		opts.Mode = model.ModeStructural
	}

	start := time.Now()
	data, err := readAll(ctx, input.Reader)
	if err != nil {
		return nil, err
	}
	detected, err := detect.Detect(data, opts.Standard)
	if err != nil {
		return &TranslateResult{
			OK:       false,
			Standard: detected.Standard,
			Mode:     opts.Mode,
			Errors: []model.EDIError{{
				Severity: "error",
				Code:     "STANDARD_DETECTION_FAILED",
				Message:  err.Error(),
			}},
		}, err
	}

	doc, err := parse(ctx, bytes.NewReader(data), detected, opts)
	if err != nil {
		return nil, err
	}
	doc.Metadata.InputName = input.Name
	doc.Metadata.ParseMs = time.Since(start).Milliseconds()
	doc.Metadata.EffectiveMode = opts.Mode

	result := &TranslateResult{
		OK:       len(doc.Errors) == 0,
		Standard: detected.Standard,
		Mode:     opts.Mode,
		Warnings: doc.Warnings,
		Errors:   doc.Errors,
		Metadata: doc.Metadata,
	}
	envelopeWarnings, envelopeErrors := schemavalidate.Envelope(doc)
	result.Warnings = appendWarnings(result.Warnings, envelopeWarnings)
	result.Errors = appendErrors(result.Errors, envelopeErrors)
	result.OK = len(result.Errors) == 0
	result.DocumentType = firstDocumentType(doc)

	if len(result.Errors) > 0 && !opts.AllowPartial {
		if opts.Mode == model.ModeAnnotated {
			result.Result = s.annotatedResult(doc, result, opts)
		} else {
			result.Result = jsonout.Structural(doc)
		}
		return result, nil
	}

	switch opts.Mode {
	case model.ModeStructural:
		result.Result = jsonout.Structural(doc)
	case model.ModeAnnotated:
		result.Result = s.annotatedResult(doc, result, opts)
	case model.ModeSemantic:
		loaded, err := s.Schemas.Resolve(opts.SchemaID, opts.SchemaPath)
		if err != nil {
			result.OK = false
			result.Errors = append(result.Errors, model.EDIError{
				Severity: "error",
				Code:     "SCHEMA_LOAD_FAILED",
				Message:  err.Error(),
			})
			return result, nil
		}
		result.Metadata.SchemaID = loaded.ID
		schemaWarnings, schemaErrors := schemavalidate.Schema(doc, loaded)
		result.Warnings = append(result.Warnings, schemaWarnings...)
		result.Errors = append(result.Errors, schemaErrors...)
		mapped, mapWarnings, mapErrors := mapping.Map(doc, loaded)
		result.Result = mapped
		result.Warnings = append(result.Warnings, mapWarnings...)
		result.Errors = append(result.Errors, mapErrors...)
		result.OK = len(result.Errors) == 0
	default:
		result.OK = false
		result.Errors = append(result.Errors, model.EDIError{
			Severity: "error",
			Code:     "UNSUPPORTED_MODE",
			Message:  fmt.Sprintf("unsupported translation mode %q", opts.Mode),
		})
	}

	return result, nil
}

func (s *Service) annotatedResult(doc *model.Document, result *TranslateResult, opts TranslateOptions) any {
	var loaded *schema.Schema
	if opts.SchemaID != "" || opts.SchemaPath != "" {
		var err error
		loaded, err = s.Schemas.Resolve(opts.SchemaID, opts.SchemaPath)
		if err != nil {
			result.OK = false
			result.Errors = append(result.Errors, model.EDIError{
				Severity: "error",
				Code:     "SCHEMA_LOAD_FAILED",
				Message:  err.Error(),
			})
			result.Metadata = doc.Metadata
			return jsonout.Annotated(doc, nil)
		}
		doc.Metadata.SchemaID = loaded.ID
		result.Metadata = doc.Metadata
	}
	return jsonout.Annotated(doc, loaded)
}

func (s *Service) Validate(ctx context.Context, input Input, opts ValidateOptions) (*ValidateResult, error) {
	mode := model.ModeStructural
	result, err := s.Translate(ctx, input, TranslateOptions{
		Standard:     opts.Standard,
		Mode:         mode,
		SchemaPath:   opts.SchemaPath,
		SchemaID:     opts.SchemaID,
		AllowPartial: true,
	})
	if result == nil {
		return nil, err
	}
	if opts.SchemaPath != "" || opts.SchemaID != "" {
		loaded, loadErr := s.Schemas.Resolve(opts.SchemaID, opts.SchemaPath)
		if loadErr != nil {
			result.Errors = append(result.Errors, model.EDIError{
				Severity: "error",
				Code:     "SCHEMA_LOAD_FAILED",
				Message:  loadErr.Error(),
			})
		} else if doc, ok := result.Result.(*model.Document); ok {
			result.Metadata.SchemaID = loaded.ID
			schemaWarnings, schemaErrors := schemavalidate.Schema(doc, loaded)
			result.Warnings = append(result.Warnings, schemaWarnings...)
			result.Errors = append(result.Errors, schemaErrors...)
		}
	}
	return &ValidateResult{
		OK:       len(result.Errors) == 0,
		Standard: result.Standard,
		Warnings: result.Warnings,
		Errors:   result.Errors,
		Metadata: result.Metadata,
	}, err
}

func parse(ctx context.Context, input io.Reader, detected detect.Result, opts TranslateOptions) (*model.Document, error) {
	switch detected.Standard {
	case model.StandardX12:
		return x12.ParseReader(ctx, input, x12.Options{
			Delimiters:     detected.Delimiters,
			IncludeRaw:     opts.IncludeRaw,
			IncludeOffsets: opts.IncludeOffsets,
		})
	case model.StandardEDIFACT:
		return edifact.ParseReader(ctx, input, edifact.Options{
			Delimiters:     detected.Delimiters,
			IncludeRaw:     opts.IncludeRaw,
			IncludeOffsets: opts.IncludeOffsets,
		})
	default:
		return nil, fmt.Errorf("unsupported standard %q", detected.Standard)
	}
}

func readAll(ctx context.Context, r io.Reader) ([]byte, error) {
	type result struct {
		data []byte
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		data, err := io.ReadAll(r)
		ch <- result{data: data, err: err}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-ch:
		return result.data, result.err
	}
}

func firstDocumentType(doc *model.Document) string {
	for _, interchange := range doc.Interchanges {
		for _, group := range interchange.Groups {
			for _, tx := range group.Transactions {
				if tx.Type != "" {
					return tx.Type
				}
			}
		}
		for _, msg := range interchange.Messages {
			if msg.Type != "" {
				return msg.Type
			}
		}
	}
	return ""
}

func appendWarnings(existing []model.EDIWarning, additions []model.EDIWarning) []model.EDIWarning {
	for _, warning := range additions {
		if hasWarning(existing, warning) {
			continue
		}
		existing = append(existing, warning)
	}
	return existing
}

func appendErrors(existing []model.EDIError, additions []model.EDIError) []model.EDIError {
	for _, ediErr := range additions {
		if hasError(existing, ediErr) {
			continue
		}
		existing = append(existing, ediErr)
	}
	return existing
}

func hasWarning(warnings []model.EDIWarning, candidate model.EDIWarning) bool {
	for _, warning := range warnings {
		if warning.Code == candidate.Code && warning.Segment == candidate.Segment && warning.Element == candidate.Element {
			return true
		}
	}
	return false
}

func hasError(errors []model.EDIError, candidate model.EDIError) bool {
	for _, ediErr := range errors {
		if ediErr.Code == candidate.Code && ediErr.Segment == candidate.Segment && ediErr.Element == candidate.Element {
			return true
		}
	}
	return false
}
