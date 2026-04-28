package translate

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/johnmonarch/ediforge/internal/model"
)

func TestTranslateAnnotatedWithSchemaAddsMetadata(t *testing.T) {
	t.Parallel()

	schemaPath := "../../schemas/examples/x12-850-basic.json"
	input := schemaExampleInput(t, schemaPath)
	service := NewService()

	annotated, err := service.Translate(context.Background(), Input{Reader: strings.NewReader(input)}, TranslateOptions{
		Mode:       model.ModeAnnotated,
		SchemaPath: schemaPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !annotated.OK {
		t.Fatalf("OK = false, errors = %+v", annotated.Errors)
	}

	structural, err := service.Translate(context.Background(), Input{Reader: strings.NewReader(input)}, TranslateOptions{
		Mode: model.ModeStructural,
	})
	if err != nil {
		t.Fatal(err)
	}
	if stableJSON(t, annotated.Result) == stableJSON(t, structural.Result) {
		t.Fatal("annotated output matched structural output")
	}

	root := jsonMap(t, annotated.Result)
	metadata := childMap(t, root, "metadata")
	if got := metadata["schemaId"]; got != "x12-850-basic" {
		t.Fatalf("metadata.schemaId = %v, want x12-850-basic", got)
	}

	beg := findTransactionSegment(t, root, "BEG")
	if got := beg["purpose"]; got != "purchase_order_header" {
		t.Fatalf("BEG purpose = %v, want purchase_order_header", got)
	}
	if got := beg["name"]; got != "Purchase Order Header" {
		t.Fatalf("BEG name = %v, want Purchase Order Header", got)
	}
	maps := childMap(t, beg, "maps")
	if got := maps["BEG03"]; got != "purchaseOrderNumber" {
		t.Fatalf("BEG03 map = %v, want purchaseOrderNumber", got)
	}

	beg03 := findElement(t, beg, "BEG03")
	if got := beg03["target"]; got != "purchaseOrderNumber" {
		t.Fatalf("BEG03 target = %v, want purchaseOrderNumber", got)
	}
	if got := beg03["name"]; got != "Purchase Order Number" {
		t.Fatalf("BEG03 name = %v, want Purchase Order Number", got)
	}

	n101 := findElement(t, findTransactionSegment(t, root, "N1"), "N101")
	if got := n101["target"]; got != "parties[].role" {
		t.Fatalf("N101 target = %v, want parties[].role", got)
	}
	if got := n101["name"]; got != "Role" {
		t.Fatalf("N101 name = %v, want Role", got)
	}
}

func TestTranslateAnnotatedWithoutSchemaAddsElementIDs(t *testing.T) {
	t.Parallel()

	input := schemaExampleInput(t, "../../schemas/examples/x12-850-basic.json")
	result, err := NewService().Translate(context.Background(), Input{Reader: strings.NewReader(input)}, TranslateOptions{
		Mode: model.ModeAnnotated,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("OK = false, errors = %+v", result.Errors)
	}

	beg := findTransactionSegment(t, jsonMap(t, result.Result), "BEG")
	beg01 := findElement(t, beg, "BEG01")
	if got := beg01["id"]; got != "BEG01" {
		t.Fatalf("BEG01 id = %v, want BEG01", got)
	}
	if _, ok := beg01["target"]; ok {
		t.Fatalf("BEG01 target = %v, want no schema target", beg01["target"])
	}
}

func TestTranslateAnnotatedWithSchemaIDUsesBundledSchema(t *testing.T) {
	t.Parallel()

	input := schemaExampleInput(t, "../../schemas/examples/x12-850-basic.json")
	service := NewService()
	service.Schemas.Roots = []string{"../../schemas/examples"}

	result, err := service.Translate(context.Background(), Input{Reader: strings.NewReader(input)}, TranslateOptions{
		Mode:     model.ModeAnnotated,
		SchemaID: "x12-850-basic",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("OK = false, errors = %+v", result.Errors)
	}

	root := jsonMap(t, result.Result)
	metadata := childMap(t, root, "metadata")
	if got := metadata["schemaId"]; got != "x12-850-basic" {
		t.Fatalf("metadata.schemaId = %v, want x12-850-basic", got)
	}
	if got := findTransactionSegment(t, root, "BEG")["purpose"]; got != "purchase_order_header" {
		t.Fatalf("BEG purpose = %v, want purchase_order_header", got)
	}
}

func jsonMap(t *testing.T, value any) map[string]any {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func childMap(t *testing.T, parent map[string]any, key string) map[string]any {
	t.Helper()

	child, ok := parent[key].(map[string]any)
	if !ok {
		t.Fatalf("%s = %#v, want object", key, parent[key])
	}
	return child
}

func findTransactionSegment(t *testing.T, root map[string]any, tag string) map[string]any {
	t.Helper()

	interchanges := childSlice(t, root, "interchanges")
	for _, interchangeValue := range interchanges {
		interchange, ok := interchangeValue.(map[string]any)
		if !ok {
			t.Fatalf("interchange = %#v, want object", interchangeValue)
		}
		for _, groupValue := range childSlice(t, interchange, "groups") {
			group, ok := groupValue.(map[string]any)
			if !ok {
				t.Fatalf("group = %#v, want object", groupValue)
			}
			for _, txValue := range childSlice(t, group, "transactions") {
				tx, ok := txValue.(map[string]any)
				if !ok {
					t.Fatalf("transaction = %#v, want object", txValue)
				}
				for _, segmentValue := range childSlice(t, tx, "segments") {
					segment, ok := segmentValue.(map[string]any)
					if !ok {
						t.Fatalf("segment = %#v, want object", segmentValue)
					}
					if segment["tag"] == tag {
						return segment
					}
				}
			}
		}
	}
	t.Fatalf("segment %s not found", tag)
	return nil
}

func findElement(t *testing.T, segment map[string]any, id string) map[string]any {
	t.Helper()

	for _, elementValue := range childSlice(t, segment, "elements") {
		element, ok := elementValue.(map[string]any)
		if !ok {
			t.Fatalf("element = %#v, want object", elementValue)
		}
		if element["id"] == id {
			return element
		}
	}
	t.Fatalf("element %s not found in segment %#v", id, segment["tag"])
	return nil
}

func childSlice(t *testing.T, parent map[string]any, key string) []any {
	t.Helper()

	child, ok := parent[key].([]any)
	if !ok {
		t.Fatalf("%s = %#v, want array", key, parent[key])
	}
	return child
}
