package validate

import (
	"fmt"

	"github.com/johnmonarch/ediforge/internal/model"
)

func Envelope(doc *model.Document) ([]model.EDIWarning, []model.EDIError) {
	result := EnvelopeResult(doc)
	return result.WarningsAndErrors()
}

func EnvelopeResult(doc *model.Document) Result {
	var result Result
	if doc == nil {
		return result
	}
	switch doc.Standard {
	case model.StandardX12:
		validateX12Envelope(doc, &result)
	case model.StandardEDIFACT:
		validateEDIFACTEnvelope(doc, &result)
	}
	return result
}

func validateX12Envelope(doc *model.Document, result *Result) {
	for interchangeIndex, interchange := range doc.Interchanges {
		path := fmt.Sprintf("$.interchanges[%d]", interchangeIndex)
		checkEnvelopeCount(result, doc.Standard, "X12_INTERCHANGE_COUNT_MISMATCH", path+".groups", "IEA01", "IEA", declaredElement(interchange.RawEnvelope, "IEA", 1), len(interchange.Groups), "IEA01 declares %d groups but parsed %d")
		checkEnvelopeControl(result, doc.Standard, "X12_INTERCHANGE_CONTROL_NUMBER_MISMATCH", path+".controlNumber", "IEA02", "IEA", interchange.ControlNumber, declaredElement(interchange.RawEnvelope, "IEA", 2), "ISA13 does not match IEA02")
		for groupIndex, group := range interchange.Groups {
			groupPath := fmt.Sprintf("%s.groups[%d]", path, groupIndex)
			if len(interchange.Groups) == 1 {
				checkEnvelopeCount(result, doc.Standard, "X12_GROUP_COUNT_MISMATCH", groupPath+".transactions", "GE01", "GE", declaredElement(interchange.RawEnvelope, "GE", 1), len(group.Transactions), "GE01 declares %d transactions but parsed %d")
				checkEnvelopeControl(result, doc.Standard, "X12_GROUP_CONTROL_NUMBER_MISMATCH", groupPath+".controlNumber", "GE02", "GE", group.ControlNumber, declaredElement(interchange.RawEnvelope, "GE", 2), "GS06 does not match GE02")
			}
			for txIndex, tx := range group.Transactions {
				txPath := fmt.Sprintf("%s.transactions[%d]", groupPath, txIndex)
				checkEnvelopeCount(result, doc.Standard, "X12_SEGMENT_COUNT_MISMATCH", txPath+".segments", "SE01", "SE", lastElement(tx.Segments, "SE", 1), len(tx.Segments), "SE01 declares %d segments but parsed %d")
				checkEnvelopeControl(result, doc.Standard, "X12_CONTROL_NUMBER_MISMATCH", txPath+".controlNumber", "SE02", "SE", tx.ControlNumber, lastElement(tx.Segments, "SE", 2), "ST02 does not match SE02")
			}
		}
	}
}

func validateEDIFACTEnvelope(doc *model.Document, result *Result) {
	for interchangeIndex, interchange := range doc.Interchanges {
		path := fmt.Sprintf("$.interchanges[%d]", interchangeIndex)
		checkEnvelopeCount(result, doc.Standard, "EDIFACT_INTERCHANGE_COUNT_MISMATCH", path+".messages", "UNZ01", "UNZ", declaredElement(interchange.RawEnvelope, "UNZ", 1), len(interchange.Messages), "UNZ01 declares %d messages but parsed %d")
		checkEnvelopeControl(result, doc.Standard, "EDIFACT_CONTROL_REFERENCE_MISMATCH", path+".controlNumber", "UNZ02", "UNZ", interchange.ControlNumber, declaredElement(interchange.RawEnvelope, "UNZ", 2), "UNB05 does not match UNZ02")
		for msgIndex, msg := range interchange.Messages {
			msgPath := fmt.Sprintf("%s.messages[%d]", path, msgIndex)
			checkEnvelopeCount(result, doc.Standard, "EDIFACT_SEGMENT_COUNT_MISMATCH", msgPath+".segments", "UNT01", "UNT", lastElement(msg.Segments, "UNT", 1), len(msg.Segments), "UNT01 declares %d segments but parsed %d")
			checkEnvelopeControl(result, doc.Standard, "EDIFACT_MESSAGE_REFERENCE_MISMATCH", msgPath+".reference", "UNT02", "UNT", msg.Reference, lastElement(msg.Segments, "UNT", 2), "UNH01 does not match UNT02")
		}
	}
}

func checkEnvelopeCount(result *Result, standard model.Standard, code, path, element, segment, declared string, actual int, message string) {
	expected, ok := atoi(declared)
	if !ok || expected == actual {
		return
	}
	result.Add(Issue{
		Rule:     Rule{Code: code, Severity: SeverityError, Path: path},
		Message:  fmt.Sprintf(message, expected, actual),
		Standard: standard,
		Segment:  segment,
		Element:  element,
	})
}

func checkEnvelopeControl(result *Result, standard model.Standard, code, path, element, segment, open, close, message string) {
	if open == "" || close == "" || open == close {
		return
	}
	result.Add(Issue{
		Rule:     Rule{Code: code, Severity: SeverityError, Path: path},
		Message:  message,
		Standard: standard,
		Segment:  segment,
		Element:  element,
	})
}

func declaredElement(segments []model.Segment, tag string, element int) string {
	for i := len(segments) - 1; i >= 0; i-- {
		if segments[i].Tag == tag {
			return model.ElementValue(segments[i].Elements, element)
		}
	}
	return ""
}

func lastElement(segments []model.Segment, tag string, element int) string {
	return declaredElement(segments, tag, element)
}

func atoi(value string) (int, bool) {
	var n int
	_, err := fmt.Sscanf(value, "%d", &n)
	return n, err == nil
}
