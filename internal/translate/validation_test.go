package translate

import (
	"context"
	"strings"
	"testing"

	"github.com/johnmonarch/ediforge/internal/model"
)

func TestSemanticMappingErrorIncludesActionablePaths(t *testing.T) {
	input := strings.Replace(schemaExampleInput(t, "../../schemas/examples/x12-850-basic.json"), "BEG*00*SA*PO-10001**20260427~", "BEG*00*SA*PO-10001**BADDATE~", 1)
	result, err := NewService().Translate(context.Background(), Input{Reader: strings.NewReader(input)}, TranslateOptions{
		Mode:       model.ModeSemantic,
		SchemaPath: "../../schemas/examples/x12-850-basic.json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.OK {
		t.Fatalf("OK = true, want mapping error")
	}
	if !hasErrorCode(result.Errors, "MAPPING_EXPRESSION_FAILED") {
		t.Fatalf("errors = %+v, want MAPPING_EXPRESSION_FAILED", result.Errors)
	}
	for _, err := range result.Errors {
		if err.Code != "MAPPING_EXPRESSION_FAILED" {
			continue
		}
		if !strings.Contains(err.Message, "orderDate") || !strings.Contains(err.Message, "BEG[0].BEG05") {
			t.Fatalf("message = %q, want target and source path", err.Message)
		}
		if !strings.Contains(err.Hint, "BEG[0].BEG05") {
			t.Fatalf("hint = %q, want source path", err.Hint)
		}
		return
	}
}
