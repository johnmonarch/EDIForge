package translate

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/openedi/ediforge/internal/model"
)

func TestTranslateStructuralGolden(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		inputPath    string
		goldenPath   string
		standard     model.Standard
		documentType string
	}{
		{
			name:         "x12 850",
			inputPath:    "../../testdata/x12/850-basic.edi",
			goldenPath:   "../../testdata/golden/x12-850-basic.structural.json",
			standard:     model.StandardX12,
			documentType: "850",
		},
		{
			name:         "edifact orders",
			inputPath:    "../../testdata/edifact/orders-basic.edi",
			goldenPath:   "../../testdata/golden/edifact-orders-basic.structural.json",
			standard:     model.StandardEDIFACT,
			documentType: "ORDERS",
		},
	}

	service := NewService()
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input, err := os.ReadFile(tt.inputPath)
			if err != nil {
				t.Fatal(err)
			}
			result, err := service.Translate(context.Background(), Input{Reader: strings.NewReader(string(input))}, TranslateOptions{
				Mode: model.ModeStructural,
			})
			if err != nil {
				t.Fatal(err)
			}
			if !result.OK {
				t.Fatalf("OK = false, errors = %+v", result.Errors)
			}
			if result.Standard != tt.standard {
				t.Fatalf("standard = %q, want %q", result.Standard, tt.standard)
			}
			if result.DocumentType != tt.documentType {
				t.Fatalf("documentType = %q, want %q", result.DocumentType, tt.documentType)
			}

			got := stableJSON(t, result.Result)
			want, err := os.ReadFile(tt.goldenPath)
			if err != nil {
				t.Fatal(err)
			}
			if got != string(want) {
				t.Fatalf("structural JSON mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
			}
		})
	}
}

func stableJSON(t *testing.T, value any) string {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	stripVolatileJSONFields(decoded)
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(decoded); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func stripVolatileJSONFields(value any) {
	switch typed := value.(type) {
	case map[string]any:
		delete(typed, "parseMs")
		delete(typed, "inputName")
		for _, child := range typed {
			stripVolatileJSONFields(child)
		}
	case []any:
		for _, child := range typed {
			stripVolatileJSONFields(child)
		}
	}
}
