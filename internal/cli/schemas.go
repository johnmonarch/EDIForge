package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/openedi/ediforge/internal/schema"
)

func runSchemas(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return ExitError{Code: 2, Err: fmt.Errorf("usage: edi-json schemas <list|validate>")}
	}
	switch args[0] {
	case "list":
		return runSchemasList(ctx, args[1:])
	case "validate":
		return runSchemasValidate(ctx, args[1:])
	default:
		return ExitError{Code: 2, Err: fmt.Errorf("unknown schemas command %q", args[0])}
	}
}

func runSchemasList(ctx context.Context, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	var pretty bool
	_, _, err = parseFlagSet("schemas list", args, map[string]bool{"pretty": true}, func(fs *flag.FlagSet) {
		fs.BoolVar(&pretty, "pretty", false, "pretty-print JSON")
	})
	if err != nil {
		return err
	}
	_ = ctx
	summaries, err := newServiceWithConfig(cfg).Schemas.List()
	if err != nil {
		return ExitError{Code: 3, Err: err}
	}
	return writeOutput("", summaries, pretty)
}

func runSchemasValidate(ctx context.Context, args []string) error {
	var pretty bool
	_, positionals, err := parseFlagSet("schemas validate", args, map[string]bool{"pretty": true}, func(fs *flag.FlagSet) {
		fs.BoolVar(&pretty, "pretty", false, "pretty-print JSON")
	})
	if err != nil {
		return err
	}
	_ = ctx
	if len(positionals) == 0 {
		return ExitError{Code: 2, Err: fmt.Errorf("schema path required")}
	}
	loaded, err := schema.LoadFile(positionals[0])
	if err != nil {
		return ExitError{Code: 3, Err: err}
	}
	return writeOutput("", loaded, pretty)
}
