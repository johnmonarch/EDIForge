package cli

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestTranslateSingleFileStdoutResultShape(t *testing.T) {
	isolateUserConfig(t)
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "850.edi")
	copyFixture(t, filepath.Join("..", "..", "testdata", "x12", "850-basic.edi"), inputPath)

	stdout, err := executeWithStdout(t, []string{"edi-json", "translate", inputPath})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if _, ok := result["ok"]; ok {
		t.Fatalf("single-file success should preserve result-only stdout shape, got ok envelope: %s", stdout)
	}
	if _, ok := result["files"]; ok {
		t.Fatalf("single-file success should not emit batch files envelope: %s", stdout)
	}
	if got := result["standard"]; got != "x12" {
		t.Fatalf("standard = %v, want x12", got)
	}
}

func TestTranslateDirectoryStdoutBatchResults(t *testing.T) {
	isolateUserConfig(t)
	dir := t.TempDir()
	copyFixture(t, filepath.Join("..", "..", "testdata", "x12", "850-basic.edi"), filepath.Join(dir, "a.edi"))
	nested := filepath.Join(dir, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested fixture dir: %v", err)
	}
	copyFixture(t, filepath.Join("..", "..", "testdata", "edifact", "orders-basic.edi"), filepath.Join(nested, "b.txt"))
	if err := os.WriteFile(filepath.Join(dir, "ignored.csv"), []byte("not edi"), 0o644); err != nil {
		t.Fatalf("write ignored fixture: %v", err)
	}

	stdout, err := executeWithStdout(t, []string{"edi-json", "translate", dir})
	if err != nil {
		t.Fatalf("Execute returned error: %v\n%s", err, stdout)
	}

	var batch struct {
		OK    bool `json:"ok"`
		Files []struct {
			Path         string           `json:"path"`
			OK           bool             `json:"ok"`
			Standard     string           `json:"standard"`
			DocumentType string           `json:"documentType"`
			Warnings     []map[string]any `json:"warnings"`
			Errors       []map[string]any `json:"errors"`
			Metadata     map[string]any   `json:"metadata"`
			Result       map[string]any   `json:"result"`
		} `json:"files"`
		Metadata struct {
			FileCount  int `json:"fileCount"`
			OKCount    int `json:"okCount"`
			ErrorCount int `json:"errorCount"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal([]byte(stdout), &batch); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if !batch.OK {
		t.Fatalf("batch ok = false, want true: %s", stdout)
	}
	if len(batch.Files) != 2 {
		t.Fatalf("file result count = %d, want 2: %s", len(batch.Files), stdout)
	}
	if batch.Metadata.FileCount != 2 || batch.Metadata.OKCount != 2 || batch.Metadata.ErrorCount != 0 {
		t.Fatalf("metadata = %+v, want fileCount=2 okCount=2 errorCount=0", batch.Metadata)
	}
	wantPaths := []string{"a.edi", "nested/b.txt"}
	wantStandards := []string{"x12", "edifact"}
	wantDocumentTypes := []string{"850", "ORDERS"}
	for i, file := range batch.Files {
		if file.Path != wantPaths[i] {
			t.Fatalf("files[%d].path = %q, want %q", i, file.Path, wantPaths[i])
		}
		if !file.OK {
			t.Fatalf("files[%d].ok = false, want true: %+v", i, file.Errors)
		}
		if file.Standard != wantStandards[i] {
			t.Fatalf("files[%d].standard = %q, want %q", i, file.Standard, wantStandards[i])
		}
		if file.DocumentType != wantDocumentTypes[i] {
			t.Fatalf("files[%d].documentType = %q, want %q", i, file.DocumentType, wantDocumentTypes[i])
		}
		if file.Warnings == nil {
			t.Fatalf("files[%d].warnings should be present as an array", i)
		}
		if file.Errors == nil {
			t.Fatalf("files[%d].errors should be present as an array", i)
		}
		if len(file.Metadata) == 0 {
			t.Fatalf("files[%d].metadata should be present", i)
		}
		if len(file.Result) == 0 {
			t.Fatalf("files[%d].result should be present", i)
		}
	}
}

func TestTranslateDirectoryExitCodeWhenAnyFileFails(t *testing.T) {
	dir := t.TempDir()
	copyFixture(t, filepath.Join("..", "..", "testdata", "x12", "850-basic.edi"), filepath.Join(dir, "a.edi"))
	if err := os.WriteFile(filepath.Join(dir, "b.edi"), []byte("not edi"), 0o644); err != nil {
		t.Fatalf("write invalid fixture: %v", err)
	}

	stdout, err := executeWithStdout(t, []string{"edi-json", "translate", dir})
	if err == nil {
		t.Fatalf("Execute returned nil error, want exit error: %s", stdout)
	}
	if got := ExitCode(err); got != 1 {
		t.Fatalf("ExitCode = %d, want 1: %v", got, err)
	}

	var batch struct {
		OK    bool `json:"ok"`
		Files []struct {
			OK     bool             `json:"ok"`
			Errors []map[string]any `json:"errors"`
		} `json:"files"`
		Metadata struct {
			FileCount  int `json:"fileCount"`
			OKCount    int `json:"okCount"`
			ErrorCount int `json:"errorCount"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal([]byte(stdout), &batch); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if batch.OK {
		t.Fatalf("batch ok = true, want false: %s", stdout)
	}
	if batch.Metadata.FileCount != 2 || batch.Metadata.OKCount != 1 || batch.Metadata.ErrorCount != 1 {
		t.Fatalf("metadata = %+v, want fileCount=2 okCount=1 errorCount=1", batch.Metadata)
	}
	if len(batch.Files) != 2 || batch.Files[1].OK || len(batch.Files[1].Errors) == 0 {
		t.Fatalf("invalid file result was not reported: %+v", batch.Files)
	}
}

func TestTranslateUsesProjectConfigDefaultsAndSchemaPaths(t *testing.T) {
	isolateUserConfig(t)
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "850.edi")
	copyFixture(t, filepath.Join(repoRoot, "testdata", "x12", "850-basic.edi"), inputPath)
	if err := os.WriteFile(filepath.Join(dir, "edi-json.yml"), []byte(`
translation:
  defaultMode: annotated
schemas:
  paths:
    - `+filepath.Join(repoRoot, "schemas", "examples")+`
`), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatalf("restore working directory: %v", err)
		}
	})

	stdout, err := executeWithStdout(t, []string{"edi-json", "translate", inputPath, "--schema-id", "x12-850-basic"})
	if err != nil {
		t.Fatalf("Execute returned error: %v\n%s", err, stdout)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	metadata, ok := result["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("metadata = %#v, want object", result["metadata"])
	}
	if got := metadata["mode"]; got != "annotated" {
		t.Fatalf("metadata.mode = %v, want annotated", got)
	}
	if got := metadata["schemaId"]; got != "x12-850-basic" {
		t.Fatalf("metadata.schemaId = %v, want x12-850-basic", got)
	}
	beg := firstSegmentWithTag(t, result, "BEG")
	if got := beg["purpose"]; got != "purchase_order_header" {
		t.Fatalf("BEG purpose = %v, want purchase_order_header", got)
	}
}

func executeWithStdout(t *testing.T, args []string) (string, error) {
	t.Helper()
	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = writer
	execErr := Execute(context.Background(), args)
	if err := writer.Close(); err != nil {
		t.Fatalf("close stdout writer: %v", err)
	}
	os.Stdout = oldStdout
	defer reader.Close()

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return string(output), execErr
}

func isolateUserConfig(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", filepath.Join(t.TempDir(), "home"))
}

func firstSegmentWithTag(t *testing.T, root map[string]any, tag string) map[string]any {
	t.Helper()

	for _, interchangeValue := range mustSlice(t, root, "interchanges") {
		interchange := mustMap(t, interchangeValue)
		for _, groupValue := range mustSlice(t, interchange, "groups") {
			group := mustMap(t, groupValue)
			for _, txValue := range mustSlice(t, group, "transactions") {
				tx := mustMap(t, txValue)
				for _, segmentValue := range mustSlice(t, tx, "segments") {
					segment := mustMap(t, segmentValue)
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

func mustMap(t *testing.T, value any) map[string]any {
	t.Helper()

	out, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("value = %#v, want object", value)
	}
	return out
}

func mustSlice(t *testing.T, parent map[string]any, key string) []any {
	t.Helper()

	out, ok := parent[key].([]any)
	if !ok {
		t.Fatalf("%s = %#v, want array", key, parent[key])
	}
	return out
}

func copyFixture(t *testing.T, source string, destination string) {
	t.Helper()
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read fixture %s: %v", source, err)
	}
	if err := os.WriteFile(destination, data, 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", destination, err)
	}
}
