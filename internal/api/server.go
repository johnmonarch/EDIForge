package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/johnmonarch/ediforge/internal/app"
	"github.com/johnmonarch/ediforge/internal/config"
	"github.com/johnmonarch/ediforge/internal/model"
	"github.com/johnmonarch/ediforge/internal/translate"
)

type Server struct {
	translator *translate.Service
	config     config.ServerConfig
	web        http.Handler
	mux        *http.ServeMux
}

type Response[T any] struct {
	OK       bool               `json:"ok"`
	Result   T                  `json:"result,omitempty"`
	Warnings []model.EDIWarning `json:"warnings,omitempty"`
	Errors   []model.EDIError   `json:"errors,omitempty"`
	Metadata model.Metadata     `json:"metadata,omitempty"`
}

func NewServer(translator *translate.Service, cfg config.ServerConfig, web http.Handler) *Server {
	s := &Server{
		translator: translator,
		config:     cfg,
		web:        web,
		mux:        http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	var handler http.Handler = s.mux
	handler = s.recover(handler)
	handler = s.auth(handler)
	handler = s.cors(handler)
	return handler
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /api/v1/version", s.handleVersion)
	s.mux.HandleFunc("POST /api/v1/detect", s.handleDetect)
	s.mux.HandleFunc("POST /api/v1/translate", s.handleTranslate)
	s.mux.HandleFunc("POST /api/v1/validate", s.handleValidate)
	s.mux.HandleFunc("GET /api/v1/schemas", s.handleSchemasList)
	s.mux.HandleFunc("POST /api/v1/schemas/validate", s.handleSchemasValidate)
	s.mux.HandleFunc("POST /api/v1/explain", s.handleExplain)
	if s.web != nil {
		s.mux.Handle("/", s.web)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": "healthy"})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"name":    app.Name,
		"command": app.Command,
		"version": app.Version,
		"commit":  app.Commit,
		"date":    app.Date,
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, Response[any]{
		OK: false,
		Errors: []model.EDIError{{
			Severity: "error",
			Code:     code,
			Message:  message,
		}},
	})
}

func (s *Server) decode(w http.ResponseWriter, r *http.Request, target any) bool {
	limit := s.config.MaxBodyMB
	if limit <= 0 {
		limit = 50
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit*1024*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return false
	}
	return true
}

func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.config.RequireToken {
			got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if got == "" {
				got = r.Header.Get("X-EDIForge-Token")
			}
			if got == "" || got != s.config.Token {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "valid API token required")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.config.CORSOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", s.config.CORSOrigin)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-EDIForge-Token")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				slog.Error("panic in API handler", "panic", fmt.Sprint(recovered), "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
