package edifact

import (
	"strings"
	"unicode"

	"github.com/openedi/ediforge/internal/model"
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
		delims.Element = "+"
	}
	if delims.Segment == "" {
		delims.Segment = "'"
	}
	if delims.Component == "" {
		delims.Component = ":"
	}
	if delims.Release == "" {
		delims.Release = "?"
	}

	var tokens []Token
	var errs []model.EDIError
	position := 0
	segmentStart := 0
	scanStart := 0
	if strings.HasPrefix(input, "UNA") && len(input) >= 9 {
		position = 1
		tok := Token{Tag: "UNA", Elements: []model.Element{}, Offset: 0, Position: position}
		if opts.IncludeRaw {
			tok.Raw = input[:9]
		}
		tokens = append(tokens, tok)
		segmentStart = 9
		scanStart = 9
	}
	var raw strings.Builder
	for i := scanStart; i < len(input); i++ {
		ch := input[i : i+1]
		if ch == delims.Release {
			if i+1 >= len(input) {
				errs = append(errs, model.EDIError{
					Severity:   "error",
					Code:       "EDIFACT_RELEASE_AT_EOF",
					Message:    "release character appears at end of input",
					Standard:   string(model.StandardEDIFACT),
					ByteOffset: int64(i),
				})
				break
			}
			raw.WriteByte(input[i])
			i++
			raw.WriteByte(input[i])
			continue
		}
		if ch != delims.Segment {
			raw.WriteByte(input[i])
			continue
		}
		segmentText := strings.Trim(raw.String(), "\r\n\t ")
		if segmentText != "" {
			position++
			tok, tokErrs := buildToken(segmentText, int64(segmentStart), position, opts)
			tokens = append(tokens, tok)
			errs = append(errs, tokErrs...)
		}
		raw.Reset()
		segmentStart = i + len(delims.Segment)
	}
	if strings.TrimSpace(raw.String()) != "" {
		position++
		tok, tokErrs := buildToken(strings.TrimSpace(raw.String()), int64(segmentStart), position, opts)
		tokens = append(tokens, tok)
		errs = append(errs, tokErrs...)
		errs = append(errs, model.EDIError{
			Severity:        "error",
			Code:            "EDIFACT_MISSING_SEGMENT_TERMINATOR",
			Message:         "input ended before the final EDIFACT segment terminator",
			Standard:        string(model.StandardEDIFACT),
			Segment:         tok.Tag,
			SegmentPosition: tok.Position,
			ByteOffset:      tok.Offset,
		})
	}
	return tokens, errs
}

func buildToken(raw string, offset int64, position int, opts Options) (Token, []model.EDIError) {
	var errs []model.EDIError
	fields := splitReleased(raw, opts.Delimiters.Element, opts.Delimiters.Release)
	tag := strings.TrimSpace(fields[0])
	elements := make([]model.Element, 0, len(fields)-1)
	for idx, field := range fields[1:] {
		value := unreleased(field, opts.Delimiters.Release)
		el := model.Element{Index: idx + 1, Value: value}
		comps := splitReleased(field, opts.Delimiters.Component, opts.Delimiters.Release)
		if len(comps) > 1 {
			el.Components = make([]string, 0, len(comps))
			for _, comp := range comps {
				el.Components = append(el.Components, unreleased(comp, opts.Delimiters.Release))
			}
		}
		elements = append(elements, el)
	}
	if !validTag(tag) {
		errs = append(errs, model.EDIError{
			Severity:        "error",
			Code:            "EDIFACT_INVALID_SEGMENT_TAG",
			Message:         "segment tag must be 3 uppercase alphanumeric characters",
			Standard:        string(model.StandardEDIFACT),
			Segment:         tag,
			SegmentPosition: position,
			ByteOffset:      offset,
		})
	}
	tok := Token{Tag: tag, Elements: elements, Offset: offset, Position: position}
	if opts.IncludeRaw {
		tok.Raw = raw
	}
	return tok, errs
}

func splitReleased(s, sep, release string) []string {
	if sep == "" {
		return []string{s}
	}
	var parts []string
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		ch := s[i : i+1]
		if release != "" && ch == release && i+1 < len(s) {
			b.WriteByte(s[i])
			i++
			b.WriteByte(s[i])
			continue
		}
		if ch == sep {
			parts = append(parts, b.String())
			b.Reset()
			continue
		}
		b.WriteByte(s[i])
	}
	parts = append(parts, b.String())
	return parts
}

func unreleased(s, release string) string {
	if release == "" || !strings.Contains(s, release) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i:i+1] == release && i+1 < len(s) {
			i++
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func validTag(tag string) bool {
	if len(tag) != 3 {
		return false
	}
	for _, r := range tag {
		if !unicode.IsUpper(r) && !unicode.IsDigit(r) {
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
