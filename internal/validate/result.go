package validate

import (
	"fmt"

	"github.com/johnmonarch/ediforge/internal/model"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Rule struct {
	Code        string
	Severity    Severity
	Path        string
	Description string
}

type Issue struct {
	Rule
	Message         string
	Standard        model.Standard
	Segment         string
	SegmentPosition int
	Element         string
	ByteOffset      int64
	Hint            string
}

type Result struct {
	Issues []Issue
}

func (r *Result) Add(issue Issue) {
	if issue.Severity == "" {
		issue.Severity = SeverityError
	}
	if issue.Hint == "" && issue.Path != "" {
		issue.Hint = fmt.Sprintf("Validation path: %s", issue.Path)
	}
	r.Issues = append(r.Issues, issue)
}

func (r Result) WarningsAndErrors() ([]model.EDIWarning, []model.EDIError) {
	var warnings []model.EDIWarning
	var errors []model.EDIError
	for _, issue := range r.Issues {
		switch issue.Severity {
		case SeverityWarning:
			warnings = append(warnings, issue.Warning())
		default:
			errors = append(errors, issue.Error())
		}
	}
	return warnings, errors
}

func (i Issue) Error() model.EDIError {
	return model.EDIError{
		Severity:        string(SeverityError),
		Code:            i.Code,
		Message:         i.Message,
		Standard:        string(i.Standard),
		Segment:         i.Segment,
		SegmentPosition: i.SegmentPosition,
		Element:         i.Element,
		ByteOffset:      i.ByteOffset,
		Hint:            i.Hint,
	}
}

func (i Issue) Warning() model.EDIWarning {
	return model.EDIWarning{
		Severity:        string(SeverityWarning),
		Code:            i.Code,
		Message:         i.Message,
		Standard:        string(i.Standard),
		Segment:         i.Segment,
		SegmentPosition: i.SegmentPosition,
		Element:         i.Element,
		ByteOffset:      i.ByteOffset,
		Hint:            i.Hint,
	}
}
