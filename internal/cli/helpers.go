package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/openedi/ediforge/internal/model"
	"github.com/openedi/ediforge/internal/translate"
)

func parseFlagSet(name string, args []string, boolFlags map[string]bool, define func(*flag.FlagSet)) (*flag.FlagSet, []string, error) {
	flagArgs, positionals := normalizeArgs(args, boolFlags)
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	define(fs)
	if err := fs.Parse(flagArgs); err != nil {
		return nil, nil, ExitError{Code: 2, Err: err}
	}
	return fs, positionals, nil
}

func normalizeArgs(args []string, boolFlags map[string]bool) ([]string, []string) {
	var flags []string
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positionals = append(positionals, arg)
			continue
		}
		flags = append(flags, arg)
		name := strings.TrimLeft(arg, "-")
		if idx := strings.Index(name, "="); idx >= 0 {
			name = name[:idx]
		}
		if strings.Contains(arg, "=") || boolFlags[name] {
			continue
		}
		if i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return flags, positionals
}

func standardFlag(value string) model.Standard {
	switch strings.ToLower(value) {
	case "", "auto":
		return model.StandardAuto
	case "x12":
		return model.StandardX12
	case "edifact":
		return model.StandardEDIFACT
	default:
		return model.Standard(value)
	}
}

func modeFlag(value string) model.Mode {
	switch strings.ToLower(value) {
	case "", "structural":
		return model.ModeStructural
	case "annotated":
		return model.ModeAnnotated
	case "semantic":
		return model.ModeSemantic
	default:
		return model.Mode(value)
	}
}

func readInput(path string) (translate.Input, error) {
	if path == "" || path == "-" {
		return translate.Input{Name: "stdin", Reader: os.Stdin}, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return translate.Input{}, ExitError{Code: 4, Err: err}
	}
	info, _ := file.Stat()
	size := int64(0)
	if info != nil {
		size = info.Size()
	}
	return translate.Input{Name: path, Reader: file, Size: size}, nil
}

func writeOutput(path string, value any, pretty bool) error {
	var writer io.Writer = os.Stdout
	var file *os.File
	if path != "" {
		var err error
		file, err = os.Create(path)
		if err != nil {
			return ExitError{Code: 4, Err: err}
		}
		defer file.Close()
		writer = file
	}
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	if pretty {
		encoder.SetIndent("", "  ")
	}
	if err := encoder.Encode(value); err != nil {
		return ExitError{Code: 5, Err: err}
	}
	return nil
}

func errSummary(errors []model.EDIError) error {
	if len(errors) == 0 {
		return fmt.Errorf("validation failed")
	}
	return fmt.Errorf("%s: %s", errors[0].Code, errors[0].Message)
}
