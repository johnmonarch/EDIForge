package translate

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openedi/ediforge/internal/model"
)

func TestExampleStarterPacksTranslate(t *testing.T) {
	t.Parallel()

	paths, err := filepath.Glob("../../schemas/examples/*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no schema examples found")
	}

	for _, path := range paths {
		path := path
		payload := readStarterPack(t, path)
		if payload.ExampleInput == "" {
			continue
		}
		t.Run(payload.ID, func(t *testing.T) {
			t.Parallel()

			result, err := NewService().Translate(context.Background(), Input{Reader: strings.NewReader(payload.ExampleInput)}, TranslateOptions{
				Mode:       model.ModeSemantic,
				SchemaPath: path,
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
			if payload.DocumentType != "" && mapped["documentType"] != payload.DocumentType {
				t.Fatalf("documentType = %v, want %s", mapped["documentType"], payload.DocumentType)
			}
			if mapped["sourceType"] == "" {
				t.Fatalf("sourceType missing in %#v", mapped)
			}
		})
	}
}

func readStarterPack(t *testing.T, path string) starterPack {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var payload starterPack
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ID == "" {
		t.Fatalf("%s did not define id", path)
	}
	return payload
}

type starterPack struct {
	ID           string `json:"id"`
	DocumentType string `json:"documentType"`
	ExampleInput string `json:"exampleInput"`
}
