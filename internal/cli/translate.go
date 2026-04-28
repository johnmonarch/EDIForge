package cli

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/johnmonarch/ediforge/internal/model"
	"github.com/johnmonarch/ediforge/internal/translate"
)

type translateBatchResult struct {
	OK       bool                   `json:"ok"`
	Input    string                 `json:"input"`
	Mode     model.Mode             `json:"mode"`
	Files    []translateBatchFile   `json:"files"`
	Warnings []model.EDIWarning     `json:"warnings,omitempty"`
	Errors   []model.EDIError       `json:"errors,omitempty"`
	Metadata translateBatchMetadata `json:"metadata"`
}

type translateBatchMetadata struct {
	FileCount  int `json:"fileCount"`
	OKCount    int `json:"okCount"`
	ErrorCount int `json:"errorCount"`
}

type translateBatchFile struct {
	Path         string             `json:"path"`
	OK           bool               `json:"ok"`
	Standard     model.Standard     `json:"standard"`
	DocumentType string             `json:"documentType,omitempty"`
	Warnings     []model.EDIWarning `json:"warnings"`
	Errors       []model.EDIError   `json:"errors"`
	Metadata     model.Metadata     `json:"metadata"`
	Result       any                `json:"result"`
}

func runTranslate(ctx context.Context, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	var standard string
	mode := cfg.Translation.DefaultMode
	var schemaPath string
	var schemaID string
	var output string
	var pretty bool
	var compact bool
	includeRaw := cfg.Translation.IncludeRawSegments
	var includeOffsets bool
	var allowPartial bool
	var jsonErrors bool

	_, positionals, err := parseFlagSet("translate", args, map[string]bool{
		"pretty": true, "compact": true, "include-raw": true, "include-offsets": true, "allow-partial": true, "json-errors": true, "no-store": true,
	}, func(fs *flag.FlagSet) {
		fs.StringVar(&standard, "standard", "auto", "auto, x12, or edifact")
		fs.StringVar(&mode, "mode", mode, "structural, annotated, or semantic")
		fs.StringVar(&schemaPath, "schema", "", "schema path")
		fs.StringVar(&schemaID, "schema-id", "", "schema id")
		fs.BoolVar(&pretty, "pretty", false, "pretty-print JSON")
		fs.BoolVar(&compact, "compact", false, "compact JSON")
		fs.StringVar(&output, "output", "", "output path")
		fs.BoolVar(&includeRaw, "include-raw", includeRaw, "include raw segments")
		fs.BoolVar(&includeOffsets, "include-offsets", false, "include byte offsets")
		fs.BoolVar(&allowPartial, "allow-partial", false, "return partial output on parse errors")
		fs.BoolVar(&jsonErrors, "json-errors", false, "return full response envelope when errors occur")
		fs.Bool("no-store", true, "accepted for privacy-compatible scripts; storage is disabled in MVP")
	})
	if err != nil {
		return err
	}
	inputPath := "-"
	if len(positionals) > 0 {
		inputPath = positionals[0]
	}
	modeValue := modeFlag(mode)
	opts := translate.TranslateOptions{
		Standard:       standardFlag(standard),
		Mode:           modeValue,
		SchemaPath:     schemaPath,
		SchemaID:       schemaID,
		Pretty:         pretty && !compact,
		IncludeRaw:     includeRaw,
		IncludeOffsets: includeOffsets,
		AllowPartial:   allowPartial,
	}
	if inputPath != "" && inputPath != "-" {
		info, statErr := os.Stat(inputPath)
		if statErr != nil {
			return ExitError{Code: 4, Err: statErr}
		}
		if info.IsDir() {
			return runTranslateDirectory(ctx, newServiceWithConfig(cfg), inputPath, output, pretty && !compact, opts)
		}
	}
	input, err := readInput(inputPath)
	if err != nil {
		return err
	}
	result, err := newServiceWithConfig(cfg).Translate(ctx, input, opts)
	if err != nil && result == nil {
		return ExitError{Code: 5, Err: err}
	}
	value := result.Result
	if !result.OK || jsonErrors || modeValue == model.ModeSemantic && len(result.Errors) > 0 {
		value = result
	}
	if err := writeOutput(output, value, pretty && !compact); err != nil {
		return err
	}
	if !result.OK {
		return ExitError{Code: 1, Err: errSummary(result.Errors)}
	}
	return nil
}

func runTranslateDirectory(ctx context.Context, service *translate.Service, dirPath string, output string, pretty bool, opts translate.TranslateOptions) error {
	paths, err := ediFilesInDirectory(dirPath)
	if err != nil {
		return err
	}

	batch := translateBatchResult{
		OK:    true,
		Input: dirPath,
		Mode:  opts.Mode,
		Files: make([]translateBatchFile, 0, len(paths)),
	}
	for _, path := range paths {
		fileResult := translateBatchFilePath(ctx, service, dirPath, path, opts)
		batch.Files = append(batch.Files, fileResult)
		if fileResult.OK {
			batch.Metadata.OKCount++
		} else {
			batch.OK = false
			batch.Metadata.ErrorCount++
		}
	}
	batch.Metadata.FileCount = len(batch.Files)
	if len(batch.Files) == 0 {
		batch.OK = false
		batch.Errors = append(batch.Errors, model.EDIError{
			Severity: "error",
			Code:     "NO_INPUT_FILES",
			Message:  "no EDI input files found",
		})
		batch.Metadata.ErrorCount = 1
	}

	if err := writeOutput(output, batch, pretty); err != nil {
		return err
	}
	if !batch.OK {
		return ExitError{Code: 1, Err: errSummary(batchErrors(batch))}
	}
	return nil
}

func ediFilesInDirectory(dirPath string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(dirPath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if isLikelyEDIFile(path) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, ExitError{Code: 4, Err: err}
	}
	sort.Strings(paths)
	return paths, nil
}

func isLikelyEDIFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".edi", ".x12", ".edifact", ".txt":
		return true
	default:
		return false
	}
}

func translateBatchFilePath(ctx context.Context, service *translate.Service, root string, path string, opts translate.TranslateOptions) translateBatchFile {
	displayPath := batchDisplayPath(root, path)
	file, err := os.Open(path)
	if err != nil {
		return translateBatchFile{
			Path:     displayPath,
			OK:       false,
			Standard: model.StandardUnknown,
			Warnings: []model.EDIWarning{},
			Errors: []model.EDIError{{
				Severity: "error",
				Code:     "INPUT_OPEN_FAILED",
				Message:  err.Error(),
			}},
		}
	}
	defer file.Close()

	info, _ := file.Stat()
	size := int64(0)
	if info != nil {
		size = info.Size()
	}
	result, err := service.Translate(ctx, translate.Input{
		Name:   path,
		Reader: file,
		Size:   size,
	}, opts)
	if err != nil && result == nil {
		return translateBatchFile{
			Path:     displayPath,
			OK:       false,
			Standard: model.StandardUnknown,
			Warnings: []model.EDIWarning{},
			Errors: []model.EDIError{{
				Severity: "error",
				Code:     "TRANSLATION_FAILED",
				Message:  err.Error(),
			}},
		}
	}
	return translateBatchFile{
		Path:         displayPath,
		OK:           result.OK,
		Standard:     result.Standard,
		DocumentType: result.DocumentType,
		Warnings:     batchWarnings(result.Warnings),
		Errors:       batchErrorsForFile(result.Errors),
		Metadata:     result.Metadata,
		Result:       result.Result,
	}
}

func batchDisplayPath(root string, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func batchErrors(batch translateBatchResult) []model.EDIError {
	if len(batch.Errors) > 0 {
		return batch.Errors
	}
	for _, file := range batch.Files {
		if len(file.Errors) > 0 {
			return file.Errors
		}
	}
	return nil
}

func batchWarnings(warnings []model.EDIWarning) []model.EDIWarning {
	if len(warnings) == 0 {
		return []model.EDIWarning{}
	}
	return warnings
}

func batchErrorsForFile(errors []model.EDIError) []model.EDIError {
	if len(errors) == 0 {
		return []model.EDIError{}
	}
	return errors
}
