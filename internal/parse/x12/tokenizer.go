package x12

import (
	"io"
	"strings"
	"unicode"

	"github.com/johnmonarch/ediforge/internal/model"
)

type Token struct {
	Tag      string
	Elements []model.Element
	Raw      string
	Offset   int64
	Position int
}

type Options struct {
	Delimiters     model.Delimiters
	IncludeRaw     bool
	IncludeOffsets bool
}

func Tokenize(input string, opts Options) ([]Token, []model.EDIError) {
	return TokenizeReader(strings.NewReader(input), opts)
}

func TokenizeReader(r io.Reader, opts Options) ([]Token, []model.EDIError) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, []model.EDIError{{
			Severity: "error",
			Code:     "X12_READ_FAILED",
			Message:  err.Error(),
			Standard: string(model.StandardX12),
		}}
	}
	input := string(data)
	delims := effectiveDelimiters(input, opts.Delimiters)
	return tokenizeString(input, opts, delims)
}

func effectiveDelimiters(input string, delims model.Delimiters) model.Delimiters {
	if delims.Element == "" {
		if len(input) > 3 && strings.HasPrefix(input, "ISA") {
			delims.Element = input[3:4]
		} else {
			delims.Element = "*"
		}
	}
	if delims.Segment == "" {
		if len(input) >= 106 && strings.HasPrefix(input, "ISA") {
			delims.Segment = input[105:106]
		} else {
			delims.Segment = "~"
		}
	}
	if delims.Component == "" {
		if len(input) >= 105 && strings.HasPrefix(input, "ISA") {
			delims.Component = input[104:105]
		} else {
			delims.Component = ">"
		}
	}
	if delims.Repetition == "" {
		delims.Repetition = "^"
		if len(input) >= 83 && strings.HasPrefix(input, "ISA") {
			if repetition := input[82:83]; repetition != "U" && repetition != "^" {
				delims.Repetition = repetition
			}
		}
	}
	return delims
}

func tokenizeString(input string, opts Options, delims model.Delimiters) ([]Token, []model.EDIError) {
	var tokens []Token
	var errs []model.EDIError
	position := 0
	start := 0
	for i := 0; i <= len(input); i++ {
		atEnd := i == len(input)
		if !atEnd && input[i:i+1] != delims.Segment {
			continue
		}
		raw := input[start:i]
		offset := int64(start)
		start = i + len(delims.Segment)

		raw = strings.Trim(raw, "\r\n\t ")
		if raw == "" {
			continue
		}
		position++
		parts := strings.Split(raw, delims.Element)
		tag := strings.TrimSpace(parts[0])
		segment := model.Segment{Tag: tag, Position: position, Offset: offset}
		if opts.IncludeRaw {
			segment.Raw = raw
		}
		elements := make([]model.Element, 0, len(parts)-1)
		for idx, part := range parts[1:] {
			el := model.Element{Index: idx + 1, Value: part}
			if delims.Component != "" && part != delims.Component && strings.Contains(part, delims.Component) {
				el.Components = strings.Split(part, delims.Component)
			}
			elements = append(elements, el)
		}
		if !validTag(tag) {
			errs = append(errs, model.EDIError{
				Severity:        "error",
				Code:            "X12_INVALID_SEGMENT_TAG",
				Message:         "segment tag must be 2 or 3 alphanumeric characters",
				Standard:        string(model.StandardX12),
				Segment:         tag,
				SegmentPosition: position,
				ByteOffset:      offset,
			})
		}
		tokens = append(tokens, Token{
			Tag:      tag,
			Elements: elements,
			Raw:      segment.Raw,
			Offset:   offset,
			Position: position,
		})
	}
	return tokens, errs
}

func validTag(tag string) bool {
	if len(tag) < 2 || len(tag) > 3 {
		return false
	}
	for _, r := range tag {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func segmentFromToken(tok Token, includeRaw, includeOffsets bool) model.Segment {
	seg := model.Segment{
		Tag:      tok.Tag,
		Position: tok.Position,
		Elements: tok.Elements,
	}
	if includeRaw {
		seg.Raw = tok.Raw
	}
	if includeOffsets {
		seg.Offset = tok.Offset
	}
	return seg
}
