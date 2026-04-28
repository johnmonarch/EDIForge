package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/johnmonarch/ediforge/internal/config"
	"github.com/johnmonarch/ediforge/internal/model"
	"github.com/johnmonarch/ediforge/internal/translate"
)

func TestHandlers(t *testing.T) {
	handler := testHandler(t)
	x12Input := readFixture(t, "testdata/x12/850-basic.edi")

	t.Run("health", func(t *testing.T) {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health", nil))

		assertStatus(t, rr, http.StatusOK)
		var body map[string]any
		decodeResponse(t, rr, &body)
		if body["ok"] != true || body["status"] != "healthy" {
			t.Fatalf("response = %+v", body)
		}
	})

	t.Run("detect", func(t *testing.T) {
		rr := postJSON(t, handler, "/api/v1/detect", map[string]string{"input": x12Input})

		assertStatus(t, rr, http.StatusOK)
		var body Response[translate.DetectResult]
		decodeResponse(t, rr, &body)
		if !body.OK {
			t.Fatalf("OK = false, errors = %+v", body.Errors)
		}
		if body.Result.Standard != model.StandardX12 {
			t.Fatalf("standard = %q", body.Result.Standard)
		}
		if body.Result.Delimiters.Element != "*" || body.Result.Delimiters.Segment != "~" {
			t.Fatalf("delimiters = %+v", body.Result.Delimiters)
		}
	})

	t.Run("translate", func(t *testing.T) {
		rr := postJSON(t, handler, "/api/v1/translate", map[string]any{
			"input": x12Input,
			"mode":  string(model.ModeStructural),
		})

		assertStatus(t, rr, http.StatusOK)
		var body translate.TranslateResult
		decodeResponse(t, rr, &body)
		if !body.OK {
			t.Fatalf("OK = false, errors = %+v", body.Errors)
		}
		if body.Standard != model.StandardX12 || body.DocumentType != "850" || body.Mode != model.ModeStructural {
			t.Fatalf("result metadata = %+v", body)
		}
		if body.Metadata.Segments != 9 || body.Metadata.Transactions != 1 {
			t.Fatalf("metadata = %+v", body.Metadata)
		}
	})

	t.Run("validate", func(t *testing.T) {
		rr := postJSON(t, handler, "/api/v1/validate", map[string]string{"input": x12Input})

		assertStatus(t, rr, http.StatusOK)
		var body translate.ValidateResult
		decodeResponse(t, rr, &body)
		if !body.OK {
			t.Fatalf("OK = false, errors = %+v", body.Errors)
		}
		if body.Standard != model.StandardX12 || body.Metadata.Segments != 9 {
			t.Fatalf("response = %+v", body)
		}
	})

	t.Run("schemas", func(t *testing.T) {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/schemas", nil))

		assertStatus(t, rr, http.StatusOK)
		var body struct {
			OK     bool `json:"ok"`
			Result []struct {
				ID       string         `json:"id"`
				Standard model.Standard `json:"standard"`
			} `json:"result"`
		}
		decodeResponse(t, rr, &body)
		if !body.OK {
			t.Fatalf("OK = false")
		}
		got := map[string]model.Standard{}
		for _, summary := range body.Result {
			got[summary.ID] = summary.Standard
		}
		if got["x12-850-basic"] != model.StandardX12 {
			t.Fatalf("missing x12 schema in %+v", got)
		}
		if got["edifact-orders-basic"] != model.StandardEDIFACT {
			t.Fatalf("missing edifact schema in %+v", got)
		}
	})
}

func TestSecurityMiddleware(t *testing.T) {
	x12Input := readFixture(t, "testdata/x12/850-basic.edi")

	t.Run("requires token when configured", func(t *testing.T) {
		handler := secureTestHandler(t, config.ServerConfig{
			RequireToken: true,
			Token:        "secret",
			MaxBodyMB:    1,
		})

		rr := postJSON(t, handler, "/api/v1/detect", map[string]string{"input": x12Input})
		assertStatus(t, rr, http.StatusUnauthorized)

		data, err := json.Marshal(map[string]string{"input": x12Input})
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/detect", strings.NewReader(string(data)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer secret")
		authed := httptest.NewRecorder()
		handler.ServeHTTP(authed, req)
		assertStatus(t, authed, http.StatusOK)
	})

	t.Run("enforces body size limit", func(t *testing.T) {
		handler := secureTestHandler(t, config.ServerConfig{MaxBodyMB: 1})
		payload := strings.Repeat("A", 1024*1024+1)

		rr := postJSON(t, handler, "/api/v1/detect", map[string]string{"input": payload})

		assertStatus(t, rr, http.StatusBadRequest)
		var body Response[any]
		decodeResponse(t, rr, &body)
		if body.OK || len(body.Errors) == 0 || body.Errors[0].Code != "INVALID_JSON" {
			t.Fatalf("response = %+v", body)
		}
	})

	t.Run("cors disabled by default and opt-in only", func(t *testing.T) {
		defaultHandler := secureTestHandler(t, config.ServerConfig{MaxBodyMB: 1})
		defaultRR := httptest.NewRecorder()
		defaultHandler.ServeHTTP(defaultRR, httptest.NewRequest(http.MethodOptions, "/api/v1/detect", nil))
		if got := defaultRR.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("default CORS origin = %q, want empty", got)
		}

		corsHandler := secureTestHandler(t, config.ServerConfig{MaxBodyMB: 1, CORSOrigin: "http://localhost:5173"})
		corsRR := httptest.NewRecorder()
		corsHandler.ServeHTTP(corsRR, httptest.NewRequest(http.MethodOptions, "/api/v1/detect", nil))
		if got := corsRR.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
			t.Fatalf("configured CORS origin = %q", got)
		}
	})
}

func testHandler(t *testing.T) http.Handler {
	t.Helper()

	return secureTestHandler(t, config.ServerConfig{MaxBodyMB: 1})
}

func secureTestHandler(t *testing.T, cfg config.ServerConfig) http.Handler {
	t.Helper()

	service := translate.NewService()
	service.Schemas.Roots = []string{filepath.Join(repoRoot(t), "schemas/examples")}
	return NewServer(service, cfg, nil).Handler()
}

func postJSON(t *testing.T, handler http.Handler, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(data)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func assertStatus(t *testing.T, rr *httptest.ResponseRecorder, want int) {
	t.Helper()

	if rr.Code != want {
		t.Fatalf("status = %d, want %d, body = %s", rr.Code, want, rr.Body.String())
	}
}

func decodeResponse(t *testing.T, rr *httptest.ResponseRecorder, target any) {
	t.Helper()

	if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Fatalf("content-type = %q", got)
	}
	if err := json.NewDecoder(rr.Body).Decode(target); err != nil {
		t.Fatal(err)
	}
}

func readFixture(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(repoRoot(t), path))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}
