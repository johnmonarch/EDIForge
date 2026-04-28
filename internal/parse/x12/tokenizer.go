package x12

import (
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
	delims := opts.Delimiters
	if delims.Element == "" {
		delims.Element = "*"
	}
	if delims.Segment == "" {
		delims.Segment = "~"
	}
	if delims.Component == "" {
		delims.Component = ">"
	}

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
