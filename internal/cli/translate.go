package cli

import (
	"context"
	"flag"

	"github.com/openedi/ediforge/internal/model"
	"github.com/openedi/ediforge/internal/translate"
)

func runTranslate(ctx context.Context, args []string) error {
	var standard string
	var mode string
	var schemaPath string
	var schemaID string
	var output string
	var pretty bool
	var compact bool
	var includeRaw bool
	var includeOffsets bool
	var allowPartial bool
	var jsonErrors bool

	_, positionals, err := parseFlagSet("translate", args, map[string]bool{
		"pretty": true, "compact": true, "include-raw": true, "include-offsets": true, "allow-partial": true, "json-errors": true, "no-store": true,
	}, func(fs *flag.FlagSet) {
		fs.StringVar(&standard, "standard", "auto", "auto, x12, or edifact")
		fs.StringVar(&mode, "mode", "structural", "structural, annotated, or semantic")
		fs.StringVar(&schemaPath, "schema", "", "schema path")
		fs.StringVar(&schemaID, "schema-id", "", "schema id")
		fs.BoolVar(&pretty, "pretty", false, "pretty-print JSON")
		fs.BoolVar(&compact, "compact", false, "compact JSON")
		fs.StringVar(&output, "output", "", "output path")
		fs.BoolVar(&includeRaw, "include-raw", false, "include raw segments")
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
	input, err := readInput(inputPath)
	if err != nil {
		return err
	}
	result, err := translate.NewService().Translate(ctx, input, translate.TranslateOptions{
		Standard:       standardFlag(standard),
		Mode:           modeFlag(mode),
		SchemaPath:     schemaPath,
		SchemaID:       schemaID,
		Pretty:         pretty && !compact,
		IncludeRaw:     includeRaw,
		IncludeOffsets: includeOffsets,
		AllowPartial:   allowPartial,
	})
	if err != nil && result == nil {
		return ExitError{Code: 5, Err: err}
	}
	value := result.Result
	if !result.OK || jsonErrors || modeFlag(mode) == model.ModeSemantic && len(result.Errors) > 0 {
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
