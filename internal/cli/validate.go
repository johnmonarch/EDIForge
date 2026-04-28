package cli

import (
	"context"
	"flag"

	"github.com/johnmonarch/ediforge/internal/translate"
)

func runValidate(ctx context.Context, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	var standard string
	var schemaPath string
	var schemaID string
	var level string
	var strict bool
	var jsonOut bool
	var pretty bool

	_, positionals, err := parseFlagSet("validate", args, map[string]bool{"strict": true, "json": true, "pretty": true}, func(fs *flag.FlagSet) {
		fs.StringVar(&standard, "standard", "auto", "auto, x12, or edifact")
		fs.StringVar(&schemaPath, "schema", "", "schema path")
		fs.StringVar(&schemaID, "schema-id", "", "schema id")
		fs.StringVar(&level, "level", "syntax", "syntax, schema, or partner")
		fs.BoolVar(&strict, "strict", false, "treat warnings as failures")
		fs.BoolVar(&jsonOut, "json", false, "emit JSON validation response")
		fs.BoolVar(&pretty, "pretty", false, "pretty-print JSON")
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
	result, err := newServiceWithConfig(cfg).Validate(ctx, input, translate.ValidateOptions{
		Standard:   standardFlag(standard),
		SchemaPath: schemaPath,
		SchemaID:   schemaID,
		Level:      level,
		Strict:     strict,
	})
	if err != nil && result == nil {
		return ExitError{Code: 5, Err: err}
	}
	if jsonOut || pretty || !result.OK {
		if err := writeOutput("", result, pretty); err != nil {
			return err
		}
	}
	if !result.OK || strict && len(result.Warnings) > 0 {
		return ExitError{Code: 1, Err: errSummary(result.Errors)}
	}
	return nil
}
