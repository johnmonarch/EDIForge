package api

import (
	"net/http"
	"strings"

	"github.com/openedi/ediforge/internal/model"
	"github.com/openedi/ediforge/internal/schema"
	"github.com/openedi/ediforge/internal/translate"
)

type detectRequest struct {
	Input    string         `json:"input"`
	Standard model.Standard `json:"standard,omitempty"`
}

type translateRequest struct {
	Input    string                     `json:"input"`
	Standard model.Standard             `json:"standard,omitempty"`
	Mode     model.Mode                 `json:"mode,omitempty"`
	SchemaID string                     `json:"schemaId,omitempty"`
	Schema   string                     `json:"schema,omitempty"`
	Options  translate.TranslateOptions `json:"options,omitempty"`
}

type validateRequest struct {
	Input    string         `json:"input"`
	Standard model.Standard `json:"standard,omitempty"`
	SchemaID string         `json:"schemaId,omitempty"`
	Schema   string         `json:"schema,omitempty"`
	Level    string         `json:"level,omitempty"`
	Strict   bool           `json:"strict,omitempty"`
}

type schemaValidateRequest struct {
	Path string `json:"path"`
}

type explainRequest struct {
	Input    string         `json:"input"`
	Standard model.Standard `json:"standard,omitempty"`
	Segment  string         `json:"segment"`
}

func (s *Server) handleDetect(w http.ResponseWriter, r *http.Request) {
	var req detectRequest
	if !s.decode(w, r, &req) {
		return
	}
	result, err := s.translator.Detect(r.Context(), translate.Input{Reader: strings.NewReader(req.Input)}, translate.DetectOptions{Standard: req.Standard})
	resp := Response[*translate.DetectResult]{OK: err == nil, Result: result}
	if err != nil {
		resp.Errors = []model.EDIError{{Severity: "error", Code: "STANDARD_DETECTION_FAILED", Message: err.Error()}}
	}
	writeJSON(w, statusFromOK(resp.OK), resp)
}

func (s *Server) handleTranslate(w http.ResponseWriter, r *http.Request) {
	var req translateRequest
	if !s.decode(w, r, &req) {
		return
	}
	opts := req.Options
	if req.Standard != "" {
		opts.Standard = req.Standard
	}
	if req.Mode != "" {
		opts.Mode = req.Mode
	}
	if req.SchemaID != "" {
		opts.SchemaID = req.SchemaID
	}
	if req.Schema != "" {
		opts.SchemaPath = req.Schema
	}
	result, err := s.translator.Translate(r.Context(), translate.Input{Reader: strings.NewReader(req.Input)}, opts)
	if err != nil && result == nil {
		writeError(w, http.StatusInternalServerError, "TRANSLATE_FAILED", err.Error())
		return
	}
	writeJSON(w, statusFromOK(result.OK), result)
}

func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	var req validateRequest
	if !s.decode(w, r, &req) {
		return
	}
	result, err := s.translator.Validate(r.Context(), translate.Input{Reader: strings.NewReader(req.Input)}, translate.ValidateOptions{
		Standard:   req.Standard,
		SchemaID:   req.SchemaID,
		SchemaPath: req.Schema,
		Level:      req.Level,
		Strict:     req.Strict,
	})
	if err != nil && result == nil {
		writeError(w, http.StatusInternalServerError, "VALIDATE_FAILED", err.Error())
		return
	}
	writeJSON(w, statusFromOK(result.OK), result)
}

func (s *Server) handleSchemasList(w http.ResponseWriter, r *http.Request) {
	summaries, err := s.translator.Schemas.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "SCHEMA_LIST_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, Response[[]schema.Summary]{OK: true, Result: summaries})
}

func (s *Server) handleSchemasValidate(w http.ResponseWriter, r *http.Request) {
	var req schemaValidateRequest
	if !s.decode(w, r, &req) {
		return
	}
	loaded, err := schema.LoadFile(req.Path)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response[*schema.Schema]{
			OK: false,
			Errors: []model.EDIError{{
				Severity: "error",
				Code:     "SCHEMA_INVALID",
				Message:  err.Error(),
			}},
		})
		return
	}
	writeJSON(w, http.StatusOK, Response[*schema.Schema]{OK: true, Result: loaded})
}

func (s *Server) handleExplain(w http.ResponseWriter, r *http.Request) {
	var req explainRequest
	if !s.decode(w, r, &req) {
		return
	}
	result, err := s.translator.Translate(r.Context(), translate.Input{Reader: strings.NewReader(req.Input)}, translate.TranslateOptions{
		Standard:     req.Standard,
		Mode:         model.ModeStructural,
		AllowPartial: true,
	})
	if err != nil && result == nil {
		writeError(w, http.StatusInternalServerError, "EXPLAIN_FAILED", err.Error())
		return
	}
	segments := collectSegments(result.Result, strings.ToUpper(req.Segment))
	writeJSON(w, http.StatusOK, Response[[]model.Segment]{OK: len(result.Errors) == 0, Result: segments, Warnings: result.Warnings, Errors: result.Errors, Metadata: result.Metadata})
}

func statusFromOK(ok bool) int {
	if ok {
		return http.StatusOK
	}
	return http.StatusBadRequest
}

func collectSegments(result any, tag string) []model.Segment {
	doc, ok := result.(*model.Document)
	if !ok || doc == nil {
		return nil
	}
	var out []model.Segment
	for _, interchange := range doc.Interchanges {
		for _, group := range interchange.Groups {
			for _, tx := range group.Transactions {
				for _, seg := range tx.Segments {
					if tag == "" || seg.Tag == tag {
						out = append(out, seg)
					}
				}
			}
		}
		for _, msg := range interchange.Messages {
			for _, seg := range msg.Segments {
				if tag == "" || seg.Tag == tag {
					out = append(out, seg)
				}
			}
		}
	}
	return out
}
