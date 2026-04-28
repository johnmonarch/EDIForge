package validate

import (
	"fmt"

	"github.com/openedi/ediforge/internal/model"
	"github.com/openedi/ediforge/internal/schema"
)

func Schema(doc *model.Document, s *schema.Schema) ([]model.EDIWarning, []model.EDIError) {
	if s == nil {
		return nil, nil
	}
	segments := flattenSegments(doc)
	counts := map[string]int{}
	for _, segment := range segments {
		counts[segment.Tag]++
	}

	var warnings []model.EDIWarning
	var errors []model.EDIError
	if s.Standard != "" && doc.Standard != "" && s.Standard != doc.Standard {
		errors = append(errors, model.EDIError{
			Severity: "error",
			Code:     "SCHEMA_STANDARD_MISMATCH",
			Message:  fmt.Sprintf("schema standard %q does not match document standard %q", s.Standard, doc.Standard),
			Standard: string(doc.Standard),
		})
	}
	if s.Transaction != "" && !hasTransaction(doc, s.Transaction) {
		errors = append(errors, model.EDIError{
			Severity: "error",
			Code:     "SCHEMA_TRANSACTION_MISMATCH",
			Message:  fmt.Sprintf("schema expects X12 transaction %q", s.Transaction),
			Standard: string(doc.Standard),
		})
	}
	if s.Message != "" && !hasMessage(doc, s.Message) {
		errors = append(errors, model.EDIError{
			Severity: "error",
			Code:     "SCHEMA_MESSAGE_MISMATCH",
			Message:  fmt.Sprintf("schema expects EDIFACT message %q", s.Message),
			Standard: string(doc.Standard),
		})
	}
	for _, rule := range s.Segments {
		if rule.Tag == "" {
			continue
		}
		count := counts[rule.Tag]
		if rule.Required && count == 0 {
			errors = append(errors, model.EDIError{
				Severity: "error",
				Code:     "SCHEMA_REQUIRED_SEGMENT_MISSING",
				Message:  fmt.Sprintf("required segment %s is missing", rule.Tag),
				Standard: string(doc.Standard),
				Segment:  rule.Tag,
			})
		}
		if rule.Max > 0 && count > rule.Max {
			errors = append(errors, model.EDIError{
				Severity: "error",
				Code:     "SCHEMA_SEGMENT_REPEAT_EXCEEDED",
				Message:  fmt.Sprintf("segment %s appears %d times, max is %d", rule.Tag, count, rule.Max),
				Standard: string(doc.Standard),
				Segment:  rule.Tag,
			})
		}
	}
	if len(s.Segments) == 0 {
		warnings = append(warnings, model.EDIWarning{
			Severity: "warning",
			Code:     "SCHEMA_HAS_NO_SEGMENT_RULES",
			Message:  fmt.Sprintf("schema %q has no segment rules", s.ID),
			Standard: string(doc.Standard),
		})
	}
	return warnings, errors
}

func flattenSegments(doc *model.Document) []model.Segment {
	var segments []model.Segment
	for _, interchange := range doc.Interchanges {
		for _, group := range interchange.Groups {
			for _, tx := range group.Transactions {
				segments = append(segments, tx.Segments...)
			}
		}
		for _, msg := range interchange.Messages {
			segments = append(segments, msg.Segments...)
		}
	}
	return segments
}

func hasTransaction(doc *model.Document, expected string) bool {
	for _, interchange := range doc.Interchanges {
		for _, group := range interchange.Groups {
			for _, tx := range group.Transactions {
				if tx.Type == expected {
					return true
				}
			}
		}
	}
	return false
}

func hasMessage(doc *model.Document, expected string) bool {
	for _, interchange := range doc.Interchanges {
		for _, msg := range interchange.Messages {
			if msg.Type == expected {
				return true
			}
		}
	}
	return false
}
