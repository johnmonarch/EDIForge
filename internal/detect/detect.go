package detect

import (
	"bytes"
	"errors"
	"strings"

	"github.com/openedi/ediforge/internal/model"
)

type Result struct {
	Standard   model.Standard   `json:"standard"`
	Confidence float64          `json:"confidence"`
	Version    string           `json:"version,omitempty"`
	Delimiters model.Delimiters `json:"delimiters,omitempty"`
	Hints      []string         `json:"hints,omitempty"`
}

var ErrUnknownStandard = errors.New("unable to detect EDI standard")

func Detect(data []byte, hint model.Standard) (Result, error) {
	data = trimBOM(data)
	switch hint {
	case model.StandardX12:
		return detectX12(data, true)
	case model.StandardEDIFACT:
		return detectEDIFACT(data, true)
	}

	if result, err := detectX12(data, false); err == nil {
		return result, nil
	}
	if result, err := detectEDIFACT(data, false); err == nil {
		return result, nil
	}

	return Result{Standard: model.StandardUnknown, Confidence: 0}, ErrUnknownStandard
}

func trimBOM(data []byte) []byte {
	return bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
}

func detectX12(data []byte, forced bool) (Result, error) {
	s := string(data)
	if len(s) >= 3 && strings.HasPrefix(s, "ISA") {
		delims, version := x12Delimiters(s)
		confidence := 0.98
		if !strings.Contains(s, delims.Segment+"GS"+delims.Element) && !strings.Contains(s, delims.Segment+"ST"+delims.Element) {
			confidence = 0.75
		}
		return Result{
			Standard:   model.StandardX12,
			Confidence: confidence,
			Version:    version,
			Delimiters: delims,
		}, nil
	}
	if forced {
		return Result{
			Standard:   model.StandardX12,
			Confidence: 0.5,
			Delimiters: model.Delimiters{Element: "*", Segment: "~", Component: ">", Repetition: "^"},
			Hints:      []string{"forced standard x12, but input does not start with ISA"},
		}, nil
	}
	return Result{}, ErrUnknownStandard
}

func x12Delimiters(s string) (model.Delimiters, string) {
	delims := model.Delimiters{
		Element:    "*",
		Segment:    "~",
		Component:  ">",
		Repetition: "^",
	}
	version := ""
	if len(s) > 3 {
		delims.Element = string(s[3])
	}
	if len(s) >= 106 {
		delims.Component = string(s[104])
		delims.Segment = string(s[105])
		if r := string(s[82]); r != "U" && r != "^" {
			delims.Repetition = r
		}
		if len(s) >= 85 {
			version = strings.TrimSpace(s[84:89])
		}
		return delims, version
	}
	if idx := strings.Index(s, "~"); idx >= 0 {
		delims.Segment = "~"
	} else if idx := strings.Index(s, "\n"); idx >= 0 {
		delims.Segment = "\n"
	}
	fields := strings.SplitN(s, delims.Segment, 2)
	isa := strings.Split(fields[0], delims.Element)
	if len(isa) > 16 {
		delims.Component = isa[16]
	}
	if len(isa) > 12 {
		version = strings.TrimSpace(isa[12])
	}
	return delims, version
}

func detectEDIFACT(data []byte, forced bool) (Result, error) {
	s := string(data)
	delims := model.Delimiters{
		Element:     "+",
		Segment:     "'",
		Component:   ":",
		Release:     "?",
		DecimalMark: ".",
	}
	if strings.HasPrefix(s, "UNA") && len(s) >= 9 {
		delims.Component = string(s[3])
		delims.Element = string(s[4])
		delims.DecimalMark = string(s[5])
		delims.Release = string(s[6])
		delims.Segment = string(s[8])
		if len(s) >= 12 && strings.HasPrefix(s[9:], "UNB") {
			return Result{Standard: model.StandardEDIFACT, Confidence: 0.99, Delimiters: delims}, nil
		}
		return Result{Standard: model.StandardEDIFACT, Confidence: 0.85, Delimiters: delims, Hints: []string{"UNA found; UNB was not found immediately after service string advice"}}, nil
	}
	if strings.HasPrefix(s, "UNB") {
		return Result{Standard: model.StandardEDIFACT, Confidence: 0.95, Delimiters: delims}, nil
	}
	if forced {
		return Result{
			Standard:   model.StandardEDIFACT,
			Confidence: 0.5,
			Delimiters: delims,
			Hints:      []string{"forced standard edifact, but input does not start with UNA or UNB"},
		}, nil
	}
	return Result{}, ErrUnknownStandard
}
