package cli

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/johnmonarch/ediforge/internal/model"
	"github.com/johnmonarch/ediforge/internal/translate"
)

func runExplain(ctx context.Context, args []string) error {
	var standard string
	var segment string
	var pretty bool
	_, positionals, err := parseFlagSet("explain", args, map[string]bool{"pretty": true}, func(fs *flag.FlagSet) {
		fs.StringVar(&standard, "standard", "auto", "auto, x12, or edifact")
		fs.StringVar(&segment, "segment", "", "segment tag to show")
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
	result, err := translate.NewService().Translate(ctx, input, translate.TranslateOptions{
		Standard:     standardFlag(standard),
		Mode:         model.ModeStructural,
		AllowPartial: true,
	})
	if err != nil && result == nil {
		return ExitError{Code: 5, Err: err}
	}
	doc, ok := result.Result.(*model.Document)
	if !ok {
		return ExitError{Code: 5, Err: fmt.Errorf("unexpected explain result type")}
	}
	matches := explainSegments(doc, strings.ToUpper(segment))
	if err := writeOutput("", matches, pretty); err != nil {
		return err
	}
	if len(result.Errors) > 0 {
		return ExitError{Code: 1, Err: errSummary(result.Errors)}
	}
	return nil
}

func explainSegments(doc *model.Document, tag string) []model.Segment {
	var matches []model.Segment
	for _, interchange := range doc.Interchanges {
		for _, group := range interchange.Groups {
			for _, tx := range group.Transactions {
				for _, seg := range tx.Segments {
					if tag == "" || seg.Tag == tag {
						matches = append(matches, seg)
					}
				}
			}
		}
		for _, msg := range interchange.Messages {
			for _, seg := range msg.Segments {
				if tag == "" || seg.Tag == tag {
					matches = append(matches, seg)
				}
			}
		}
	}
	return matches
}
