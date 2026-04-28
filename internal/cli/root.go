package cli

import (
	"context"
	"errors"
	"fmt"
)

type ExitError struct {
	Code int
	Err  error
}

func (e ExitError) Error() string {
	return e.Err.Error()
}

func ExitCode(err error) int {
	var exit ExitError
	if errors.As(err, &exit) {
		return exit.Code
	}
	return 5
}

func Execute(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return usage()
	}
	switch args[1] {
	case "translate":
		return runTranslate(ctx, args[2:])
	case "validate":
		return runValidate(ctx, args[2:])
	case "detect":
		return runDetect(ctx, args[2:])
	case "serve":
		return runServe(ctx, args[2:])
	case "schemas":
		return runSchemas(ctx, args[2:])
	case "explain":
		return runExplain(ctx, args[2:])
	case "version":
		return runVersion()
	case "help", "-h", "--help":
		return usage()
	default:
		return ExitError{Code: 2, Err: fmt.Errorf("unknown command %q", args[1])}
	}
}

func usage() error {
	return ExitError{Code: 2, Err: errors.New(`usage: edi-json <command> [options]

Commands:
  translate   Translate X12 or EDIFACT to JSON
  validate    Validate syntax and envelopes
  detect      Detect standard and delimiters
  serve       Run local REST API and web UI
  schemas     List or validate schema files
  explain     Show parsed segments by tag
  version     Print version`)}
}
