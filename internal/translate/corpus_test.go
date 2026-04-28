package translate

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/johnmonarch/ediforge/internal/model"
)

type corpusCatalog struct {
	Version     int             `json:"version"`
	Description string          `json:"description"`
	Fixtures    []corpusFixture `json:"fixtures"`
}

type corpusFixture struct {
	Name         string         `json:"name"`
	Path         string         `json:"path"`
	Category     string         `json:"category"`
	Standard     model.Standard `json:"standard"`
	DocumentType string         `json:"documentType"`
	OK           bool           `json:"ok"`
	Errors       []string       `json:"errors"`
	Warnings     []string       `json:"warnings"`
}

func TestPublicSafeCorpus(t *testing.T) {
	t.Parallel()

	catalogPath := filepath.Clean("../../testdata/corpus/catalog.json")
	catalog := loadCorpusCatalog(t, catalogPath)
	if len(catalog.Fixtures) == 0 {
		t.Fatal("catalog contains no fixtures")
	}

	seenCategories := map[string]bool{}
	service := NewService()
	for _, fixture := range catalog.Fixtures {
		fixture := fixture
		validateCorpusFixture(t, fixture)
		seenCategories[fixture.Category] = true

		t.Run(fixture.Name, func(t *testing.T) {
			t.Parallel()

			inputPath := filepath.Join(filepath.Dir(catalogPath), fixture.Path)
			input, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatal(err)
			}

			detected, err := service.Detect(context.Background(), Input{
				Name:   fixture.Name,
				Reader: strings.NewReader(string(input)),
			}, DetectOptions{Standard: model.StandardAuto})
			if err != nil {
				t.Fatalf("detect failed: %v", err)
			}
			if detected.Standard != fixture.Standard {
				t.Fatalf("detected standard = %q, want %q", detected.Standard, fixture.Standard)
			}

			result, err := service.Translate(context.Background(), Input{
				Name:   fixture.Name,
				Reader: strings.NewReader(string(input)),
			}, TranslateOptions{
				Mode:         model.ModeStructural,
				AllowPartial: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.OK != fixture.OK {
				t.Fatalf("OK = %v, want %v; errors = %+v", result.OK, fixture.OK, result.Errors)
			}
			if result.Standard != fixture.Standard {
				t.Fatalf("standard = %q, want %q", result.Standard, fixture.Standard)
			}
			if result.DocumentType != fixture.DocumentType {
				t.Fatalf("documentType = %q, want %q", result.DocumentType, fixture.DocumentType)
			}

			assertDiagnosticCodes(t, "errors", errorCodes(result.Errors), fixture.Errors)
			assertDiagnosticCodes(t, "warnings", warningCodes(result.Warnings), fixture.Warnings)
		})
	}

	for _, category := range []string{"valid", "malformed", "partial"} {
		if !seenCategories[category] {
			t.Fatalf("catalog missing %q fixture", category)
		}
	}
}

func loadCorpusCatalog(t *testing.T, path string) corpusCatalog {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var catalog corpusCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		t.Fatal(err)
	}
	if catalog.Version != 1 {
		t.Fatalf("catalog version = %d, want 1", catalog.Version)
	}
	return catalog
}

func validateCorpusFixture(t *testing.T, fixture corpusFixture) {
	t.Helper()

	if fixture.Name == "" {
		t.Fatal("fixture name is required")
	}
	if fixture.Path == "" {
		t.Fatalf("%s: fixture path is required", fixture.Name)
	}
	switch fixture.Category {
	case "valid", "malformed", "partial":
	default:
		t.Fatalf("%s: unsupported category %q", fixture.Name, fixture.Category)
	}
	switch fixture.Standard {
	case model.StandardX12, model.StandardEDIFACT:
	default:
		t.Fatalf("%s: unsupported standard %q", fixture.Name, fixture.Standard)
	}
	if fixture.DocumentType == "" {
		t.Fatalf("%s: documentType is required", fixture.Name)
	}
	if fixture.Category == "valid" && !fixture.OK {
		t.Fatalf("%s: valid fixtures must expect ok=true", fixture.Name)
	}
	if fixture.Category != "valid" && len(fixture.Errors) == 0 && len(fixture.Warnings) == 0 {
		t.Fatalf("%s: non-valid fixtures must assert errors or warnings", fixture.Name)
	}
}

func assertDiagnosticCodes(t *testing.T, label string, got, want []string) {
	t.Helper()

	for _, code := range want {
		if !slices.Contains(got, code) {
			t.Fatalf("missing %s code %s in %v", label, code, got)
		}
	}
	if len(want) == 0 && len(got) > 0 {
		t.Fatalf("unexpected %s codes: %v", label, got)
	}
}

func errorCodes(errs []model.EDIError) []string {
	codes := make([]string, 0, len(errs))
	for _, err := range errs {
		codes = append(codes, err.Code)
	}
	return codes
}

func warningCodes(warnings []model.EDIWarning) []string {
	codes := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		codes = append(codes, warning.Code)
	}
	return codes
}
