package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/johnmonarch/ediforge/internal/translate"
)

func runDetect(ctx context.Context, args []string) error {
	var standard string
	var jsonOut bool
	var pretty bool
	_, positionals, err := parseFlagSet("detect", args, map[string]bool{"json": true, "pretty": true}, func(fs *flag.FlagSet) {
		fs.StringVar(&standard, "standard", "auto", "auto, x12, or edifact")
		fs.BoolVar(&jsonOut, "json", false, "emit JSON")
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
	result, err := translate.NewService().Detect(ctx, input, translate.DetectOptions{Standard: standardFlag(standard)})
	if err != nil && result == nil {
		return ExitError{Code: 1, Err: err}
	}
	if jsonOut || pretty {
		if err := writeOutput("", result, pretty); err != nil {
			return err
		}
	} else {
		fmt.Printf("standard=%s confidence=%.2f segment=%q element=%q component=%q\n", result.Standard, result.Confidence, result.Delimiters.Segment, result.Delimiters.Element, result.Delimiters.Component)
	}
	if err != nil {
		return ExitError{Code: 1, Err: err}
	}
	return nil
}
