package translate

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/johnmonarch/ediforge/internal/model"
)

func TestTranslateSemanticMapsLoops(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		schemaPath string
		wantKey    string
	}{
		{name: "x12", schemaPath: "../../schemas/examples/x12-850-basic.json", wantKey: "quantity"},
		{name: "edifact", schemaPath: "../../schemas/examples/edifact-orders-basic.json", wantKey: "orderedQuantity"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := schemaExampleInput(t, tt.schemaPath)
			result, err := NewService().Translate(context.Background(), Input{Reader: strings.NewReader(input)}, TranslateOptions{
				Mode:       model.ModeSemantic,
				SchemaPath: tt.schemaPath,
			})
			if err != nil {
				t.Fatal(err)
			}
			if !result.OK {
				t.Fatalf("OK = false, errors = %+v", result.Errors)
			}
			mapped, ok := result.Result.(map[string]any)
			if !ok {
				t.Fatalf("result type = %T", result.Result)
			}
			if got := mapped["purchaseOrderNumber"]; got != "PO-10001" {
				t.Fatalf("purchaseOrderNumber = %v", got)
			}
			if got := mapped["orderDate"]; got != "2026-04-27" {
				t.Fatalf("orderDate = %v", got)
			}
			parties, ok := mapped["parties"].([]map[string]any)
			if !ok || len(parties) < 2 {
				t.Fatalf("parties = %#v", mapped["parties"])
			}
			lineItems, ok := mapped["lineItems"].([]map[string]any)
			if !ok || len(lineItems) != 1 {
				t.Fatalf("lineItems = %#v", mapped["lineItems"])
			}
			if got := lineItems[0][tt.wantKey]; got != "10" {
				t.Fatalf("line item %s = %v", tt.wantKey, got)
			}
		})
	}
}

func TestValidateWithSchemaReportsRequiredSegment(t *testing.T) {
	t.Parallel()

	schemaPath := "../../schemas/examples/x12-850-basic.json"
	input := strings.Replace(schemaExampleInput(t, schemaPath), "BEG*00*SA*PO-10001**20260427~", "", 1)
	result, err := NewService().Validate(context.Background(), Input{Reader: strings.NewReader(input)}, ValidateOptions{
		SchemaPath: schemaPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Fatalf("OK = true, want schema validation failure")
	}
	if !hasErrorCode(result.Errors, "SCHEMA_REQUIRED_SEGMENT_MISSING") {
		t.Fatalf("errors = %+v", result.Errors)
	}
}

func schemaExampleInput(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		ExampleInput string `json:"exampleInput"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ExampleInput == "" {
		t.Fatalf("%s did not define exampleInput", path)
	}
	return payload.ExampleInput
}

func hasErrorCode(errors []model.EDIError, code string) bool {
	for _, err := range errors {
		if err.Code == code {
			return true
		}
	}
	return false
}
