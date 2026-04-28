package edifact

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/johnmonarch/ediforge/internal/model"
)

func Parse(ctx context.Context, input string, opts Options) (*model.Document, error) {
	tokens, tokenizeErrs := Tokenize(input, opts)
	doc := &model.Document{
		Standard: model.StandardEDIFACT,
		Metadata: model.Metadata{
			Segments:   len(tokens),
			Delimiters: opts.Delimiters,
		},
	}
	doc.Errors = append(doc.Errors, tokenizeErrs...)

	var currentInterchange *model.Interchange
	var currentMessage *model.Message
	var unaSegment *model.Segment

	for _, tok := range tokens {
		select {
		case <-ctx.Done():
			return doc, ctx.Err()
		default:
		}

		seg := segmentFromToken(tok, opts.IncludeRaw, opts.IncludeOffsets)
		switch tok.Tag {
		case "UNA":
			unaSegment = &seg
		case "UNB":
			if currentInterchange != nil && currentInterchange.ControlNumber != "" {
				doc.Errors = append(doc.Errors, ediError("EDIFACT_UNCLOSED_INTERCHANGE", "new UNB started before previous UNZ", seg))
				closeInterchange(doc, currentInterchange, currentMessage)
			}
			currentInterchange = &model.Interchange{
				Standard:      model.StandardEDIFACT,
				SenderID:      elementComponent(tok, 2, 1),
				ReceiverID:    elementComponent(tok, 3, 1),
				ControlNumber: elementValue(tok, 5),
				RawEnvelope:   []model.Segment{seg},
			}
			if unaSegment != nil {
				currentInterchange.RawEnvelope = append([]model.Segment{*unaSegment}, currentInterchange.RawEnvelope...)
			}
			doc.Version = elementComponent(tok, 1, 2)
			currentMessage = nil
		case "UNG":
			if currentInterchange == nil {
				doc.Errors = append(doc.Errors, ediError("EDIFACT_MISSING_UNB", "UNG appeared before UNB", seg))
				currentInterchange = &model.Interchange{Standard: model.StandardEDIFACT}
			}
			currentInterchange.RawEnvelope = append(currentInterchange.RawEnvelope, seg)
		case "UNH":
			if currentInterchange == nil {
				doc.Errors = append(doc.Errors, ediError("EDIFACT_MISSING_UNB", "UNH appeared before UNB", seg))
				currentInterchange = &model.Interchange{Standard: model.StandardEDIFACT}
			}
			if currentMessage != nil {
				doc.Errors = append(doc.Errors, ediError("EDIFACT_UNCLOSED_MESSAGE", "new UNH started before previous UNT", seg))
				closeMessage(currentInterchange, currentMessage)
			}
			currentMessage = &model.Message{
				Type:            elementComponent(tok, 2, 1),
				Version:         elementComponent(tok, 2, 2),
				Release:         elementComponent(tok, 2, 3),
				ControllingOrg:  elementComponent(tok, 2, 4),
				AssociationCode: elementComponent(tok, 2, 5),
				Reference:       elementValue(tok, 1),
				Segments:        []model.Segment{seg},
			}
		case "UNT":
			if currentMessage == nil {
				doc.Errors = append(doc.Errors, ediError("EDIFACT_UNEXPECTED_UNT", "UNT appeared without an open message", seg))
				continue
			}
			currentMessage.Segments = append(currentMessage.Segments, seg)
			validateMessage(doc, currentMessage, tok)
			if currentInterchange == nil {
				currentInterchange = &model.Interchange{Standard: model.StandardEDIFACT}
			}
			closeMessage(currentInterchange, currentMessage)
			currentMessage = nil
		case "UNE":
			if currentInterchange == nil {
				doc.Errors = append(doc.Errors, ediError("EDIFACT_UNEXPECTED_UNE", "UNE appeared without an open interchange", seg))
				continue
			}
			currentInterchange.RawEnvelope = append(currentInterchange.RawEnvelope, seg)
		case "UNZ":
			if currentInterchange == nil {
				doc.Errors = append(doc.Errors, ediError("EDIFACT_UNEXPECTED_UNZ", "UNZ appeared without an open interchange", seg))
				continue
			}
			if currentMessage != nil {
				doc.Errors = append(doc.Errors, ediError("EDIFACT_MISSING_UNT", "interchange closed while a message was still open", seg))
				closeMessage(currentInterchange, currentMessage)
				currentMessage = nil
			}
			currentInterchange.RawEnvelope = append(currentInterchange.RawEnvelope, seg)
			validateInterchange(doc, currentInterchange, tok)
			doc.Interchanges = append(doc.Interchanges, *currentInterchange)
			currentInterchange = nil
		default:
			if currentMessage != nil {
				currentMessage.Segments = append(currentMessage.Segments, seg)
			} else if currentInterchange != nil {
				doc.Warnings = append(doc.Warnings, model.Warning("EDIFACT_SEGMENT_OUTSIDE_MESSAGE", "segment appeared outside an UNH/UNT message", model.StandardEDIFACT, seg))
			} else {
				doc.Errors = append(doc.Errors, ediError("EDIFACT_SEGMENT_BEFORE_UNB", "segment appeared before UNB envelope", seg))
			}
		}
	}

	if currentMessage != nil {
		doc.Errors = append(doc.Errors, model.EDIError{Severity: "error", Code: "EDIFACT_MISSING_UNT", Message: "message ended before UNT trailer", Standard: string(model.StandardEDIFACT), Segment: currentMessage.Type})
		if currentInterchange == nil {
			currentInterchange = &model.Interchange{Standard: model.StandardEDIFACT}
		}
		closeMessage(currentInterchange, currentMessage)
	}
	if currentInterchange != nil {
		doc.Errors = append(doc.Errors, model.EDIError{Severity: "error", Code: "EDIFACT_MISSING_UNZ", Message: "interchange ended before UNZ trailer", Standard: string(model.StandardEDIFACT)})
		doc.Interchanges = append(doc.Interchanges, *currentInterchange)
	}
	if len(doc.Interchanges) == 0 && len(tokens) > 0 {
		doc.Errors = append(doc.Errors, model.EDIError{Severity: "error", Code: "EDIFACT_MISSING_UNB", Message: "input did not contain a UNB envelope", Standard: string(model.StandardEDIFACT)})
	}
	summarize(doc)
	return doc, nil
}

func closeInterchange(doc *model.Document, interchange *model.Interchange, msg *model.Message) {
	if msg != nil {
		closeMessage(interchange, msg)
	}
	doc.Interchanges = append(doc.Interchanges, *interchange)
}

func closeMessage(interchange *model.Interchange, msg *model.Message) {
	msg.SegmentCount = len(msg.Segments)
	interchange.Messages = append(interchange.Messages, *msg)
}

func validateMessage(doc *model.Document, msg *model.Message, unt Token) {
	if msg.Reference != "" && elementValue(unt, 2) != "" && msg.Reference != elementValue(unt, 2) {
		err := ediError("EDIFACT_MESSAGE_REFERENCE_MISMATCH", "UNH01 does not match UNT02", segmentFromToken(unt, false, true))
		err.Element = "UNT02"
		doc.Errors = append(doc.Errors, err)
	}
	if expected, ok := atoi(elementValue(unt, 1)); ok && expected != len(msg.Segments) {
		err := ediError("EDIFACT_SEGMENT_COUNT_MISMATCH", fmt.Sprintf("UNT01 declares %d segments but parsed %d", expected, len(msg.Segments)), segmentFromToken(unt, false, true))
		err.Element = "UNT01"
		doc.Errors = append(doc.Errors, err)
	}
	if len(msg.Segments) <= 2 {
		doc.Warnings = append(doc.Warnings, model.EDIWarning{Severity: "warning", Code: "EDIFACT_EMPTY_MESSAGE", Message: "message contains no body segments", Standard: string(model.StandardEDIFACT), Segment: "UNH"})
	}
}

func validateInterchange(doc *model.Document, interchange *model.Interchange, unz Token) {
	if expected, ok := atoi(elementValue(unz, 1)); ok && expected != len(interchange.Messages) {
		err := ediError("EDIFACT_INTERCHANGE_COUNT_MISMATCH", fmt.Sprintf("UNZ01 declares %d messages but parsed %d", expected, len(interchange.Messages)), segmentFromToken(unz, false, true))
		err.Element = "UNZ01"
		doc.Errors = append(doc.Errors, err)
	}
	if interchange.ControlNumber != "" && elementValue(unz, 2) != "" && interchange.ControlNumber != elementValue(unz, 2) {
		err := ediError("EDIFACT_CONTROL_REFERENCE_MISMATCH", "UNB05 does not match UNZ02", segmentFromToken(unz, false, true))
		err.Element = "UNZ02"
		doc.Errors = append(doc.Errors, err)
	}
}

func summarize(doc *model.Document) {
	for _, interchange := range doc.Interchanges {
		doc.Metadata.Messages += len(interchange.Messages)
	}
	doc.Metadata.TranslatedBy = "EDIForge"
}

func elementValue(tok Token, index int) string {
	if index <= 0 || index > len(tok.Elements) {
		return ""
	}
	return strings.TrimSpace(tok.Elements[index-1].Value)
}

func elementComponent(tok Token, index, component int) string {
	if index <= 0 || index > len(tok.Elements) {
		return ""
	}
	el := tok.Elements[index-1]
	if component <= 0 || len(el.Components) == 0 {
		return strings.TrimSpace(el.Value)
	}
	if component > len(el.Components) {
		return ""
	}
	return strings.TrimSpace(el.Components[component-1])
}

func atoi(s string) (int, bool) {
	i, err := strconv.Atoi(strings.TrimSpace(s))
	return i, err == nil
}

func ediError(code, message string, seg model.Segment) model.EDIError {
	return model.EDIError{
		Severity:        "error",
		Code:            code,
		Message:         message,
		Standard:        string(model.StandardEDIFACT),
		Segment:         seg.Tag,
		SegmentPosition: seg.Position,
		ByteOffset:      seg.Offset,
	}
}
