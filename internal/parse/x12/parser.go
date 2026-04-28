package x12

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
		Standard: model.StandardX12,
		Metadata: model.Metadata{
			Segments:   len(tokens),
			Delimiters: opts.Delimiters,
		},
	}
	doc.Errors = append(doc.Errors, tokenizeErrs...)

	var currentInterchange *model.Interchange
	var currentGroup *model.Group
	var currentTransaction *model.Transaction

	for _, tok := range tokens {
		select {
		case <-ctx.Done():
			return doc, ctx.Err()
		default:
		}

		seg := segmentFromToken(tok, opts.IncludeRaw, opts.IncludeOffsets)
		switch tok.Tag {
		case "ISA":
			if currentInterchange != nil {
				doc.Errors = append(doc.Errors, ediError("X12_UNCLOSED_INTERCHANGE", "new ISA started before previous interchange closed", seg))
				closeInterchange(doc, currentInterchange, currentGroup, currentTransaction)
			}
			currentInterchange = &model.Interchange{
				Standard:      model.StandardX12,
				SenderID:      strings.TrimSpace(value(tok, 6)),
				ReceiverID:    strings.TrimSpace(value(tok, 8)),
				ControlNumber: strings.TrimSpace(value(tok, 13)),
				RawEnvelope:   []model.Segment{seg},
			}
			doc.Version = strings.TrimSpace(value(tok, 12))
			doc.Metadata.Delimiters = opts.Delimiters
			currentGroup = nil
			currentTransaction = nil
		case "GS":
			if currentInterchange == nil {
				doc.Errors = append(doc.Errors, ediError("X12_MISSING_ISA", "GS appeared before ISA", seg))
				currentInterchange = &model.Interchange{Standard: model.StandardX12}
			}
			if currentGroup != nil {
				doc.Errors = append(doc.Errors, ediError("X12_UNCLOSED_GROUP", "new GS started before previous GE", seg))
				currentInterchange.Groups = append(currentInterchange.Groups, *currentGroup)
			}
			currentGroup = &model.Group{
				FunctionalID:  value(tok, 1),
				Version:       value(tok, 8),
				ControlNumber: value(tok, 6),
			}
			currentInterchange.RawEnvelope = append(currentInterchange.RawEnvelope, seg)
			currentTransaction = nil
		case "ST":
			if currentGroup == nil {
				doc.Errors = append(doc.Errors, ediError("X12_MISSING_GS", "ST appeared before GS", seg))
				if currentInterchange == nil {
					currentInterchange = &model.Interchange{Standard: model.StandardX12}
				}
				currentGroup = &model.Group{}
			}
			if currentTransaction != nil {
				doc.Errors = append(doc.Errors, ediError("X12_UNCLOSED_TRANSACTION", "new ST started before previous SE", seg))
				closeTransaction(currentGroup, currentTransaction)
			}
			currentTransaction = &model.Transaction{
				Type:          value(tok, 1),
				Version:       firstNonEmpty(value(tok, 3), currentGroup.Version, doc.Version),
				ControlNumber: value(tok, 2),
				Segments:      []model.Segment{seg},
			}
		case "SE":
			if currentTransaction == nil {
				doc.Errors = append(doc.Errors, ediError("X12_UNEXPECTED_SE", "SE appeared without an open transaction", seg))
				continue
			}
			currentTransaction.Segments = append(currentTransaction.Segments, seg)
			validateTransaction(doc, currentTransaction, tok)
			closeTransaction(currentGroup, currentTransaction)
			currentTransaction = nil
		case "GE":
			if currentGroup == nil {
				doc.Errors = append(doc.Errors, ediError("X12_UNEXPECTED_GE", "GE appeared without an open functional group", seg))
				continue
			}
			if currentTransaction != nil {
				doc.Errors = append(doc.Errors, ediError("X12_MISSING_SE", "functional group closed while a transaction was still open", seg))
				closeTransaction(currentGroup, currentTransaction)
				currentTransaction = nil
			}
			validateGroup(doc, currentGroup, tok)
			if currentInterchange == nil {
				currentInterchange = &model.Interchange{Standard: model.StandardX12}
			}
			currentInterchange.RawEnvelope = append(currentInterchange.RawEnvelope, seg)
			currentInterchange.Groups = append(currentInterchange.Groups, *currentGroup)
			currentGroup = nil
		case "IEA":
			if currentInterchange == nil {
				doc.Errors = append(doc.Errors, ediError("X12_UNEXPECTED_IEA", "IEA appeared without an open interchange", seg))
				continue
			}
			if currentTransaction != nil {
				doc.Errors = append(doc.Errors, ediError("X12_MISSING_SE", "interchange closed while a transaction was still open", seg))
				if currentGroup == nil {
					currentGroup = &model.Group{}
				}
				closeTransaction(currentGroup, currentTransaction)
				currentTransaction = nil
			}
			if currentGroup != nil {
				doc.Errors = append(doc.Errors, ediError("X12_MISSING_GE", "interchange closed while a group was still open", seg))
				currentInterchange.Groups = append(currentInterchange.Groups, *currentGroup)
				currentGroup = nil
			}
			currentInterchange.RawEnvelope = append(currentInterchange.RawEnvelope, seg)
			validateInterchange(doc, currentInterchange, tok)
			doc.Interchanges = append(doc.Interchanges, *currentInterchange)
			currentInterchange = nil
		default:
			if currentTransaction != nil {
				currentTransaction.Segments = append(currentTransaction.Segments, seg)
			} else if currentInterchange != nil {
				doc.Warnings = append(doc.Warnings, model.Warning("X12_SEGMENT_OUTSIDE_TRANSACTION", "segment appeared outside an ST/SE transaction", model.StandardX12, seg))
			} else {
				doc.Errors = append(doc.Errors, ediError("X12_SEGMENT_BEFORE_ISA", "segment appeared before ISA envelope", seg))
			}
		}
	}

	if currentTransaction != nil {
		doc.Errors = append(doc.Errors, model.EDIError{
			Severity: "error",
			Code:     "X12_MISSING_SE",
			Message:  "transaction ended before SE trailer",
			Standard: string(model.StandardX12),
			Segment:  currentTransaction.Type,
		})
		if currentGroup == nil {
			currentGroup = &model.Group{}
		}
		closeTransaction(currentGroup, currentTransaction)
	}
	if currentGroup != nil {
		doc.Errors = append(doc.Errors, model.EDIError{Severity: "error", Code: "X12_MISSING_GE", Message: "functional group ended before GE trailer", Standard: string(model.StandardX12)})
		if currentInterchange == nil {
			currentInterchange = &model.Interchange{Standard: model.StandardX12}
		}
		currentInterchange.Groups = append(currentInterchange.Groups, *currentGroup)
	}
	if currentInterchange != nil {
		doc.Errors = append(doc.Errors, model.EDIError{Severity: "error", Code: "X12_MISSING_IEA", Message: "interchange ended before IEA trailer", Standard: string(model.StandardX12)})
		doc.Interchanges = append(doc.Interchanges, *currentInterchange)
	}
	if len(doc.Interchanges) == 0 && len(tokens) > 0 {
		doc.Errors = append(doc.Errors, model.EDIError{Severity: "error", Code: "X12_MISSING_ISA", Message: "input did not contain an ISA envelope", Standard: string(model.StandardX12)})
	}
	summarize(doc)
	return doc, nil
}

func closeInterchange(doc *model.Document, interchange *model.Interchange, group *model.Group, txn *model.Transaction) {
	if txn != nil {
		if group == nil {
			group = &model.Group{}
		}
		closeTransaction(group, txn)
	}
	if group != nil {
		interchange.Groups = append(interchange.Groups, *group)
	}
	doc.Interchanges = append(doc.Interchanges, *interchange)
}

func closeTransaction(group *model.Group, txn *model.Transaction) {
	txn.SegmentCount = len(txn.Segments)
	group.Transactions = append(group.Transactions, *txn)
}

func validateTransaction(doc *model.Document, txn *model.Transaction, se Token) {
	expectedControl := value(se, 2)
	if txn.ControlNumber != "" && expectedControl != "" && txn.ControlNumber != expectedControl {
		err := ediError("X12_CONTROL_NUMBER_MISMATCH", "ST02 does not match SE02", segmentFromToken(se, false, true))
		err.Element = "SE02"
		err.Hint = "Check whether the transaction was truncated or concatenated incorrectly."
		doc.Errors = append(doc.Errors, err)
	}
	if expectedCount, ok := atoi(value(se, 1)); ok && expectedCount != len(txn.Segments) {
		err := ediError("X12_SEGMENT_COUNT_MISMATCH", fmt.Sprintf("SE01 declares %d segments but parsed %d", expectedCount, len(txn.Segments)), segmentFromToken(se, false, true))
		err.Element = "SE01"
		doc.Errors = append(doc.Errors, err)
	}
	if len(txn.Segments) <= 2 {
		doc.Warnings = append(doc.Warnings, model.EDIWarning{Severity: "warning", Code: "X12_EMPTY_TRANSACTION", Message: "transaction contains no body segments", Standard: string(model.StandardX12), Segment: "ST"})
	}
}

func validateGroup(doc *model.Document, group *model.Group, ge Token) {
	if expected, ok := atoi(value(ge, 1)); ok && expected != len(group.Transactions) {
		err := ediError("X12_GROUP_COUNT_MISMATCH", fmt.Sprintf("GE01 declares %d transactions but parsed %d", expected, len(group.Transactions)), segmentFromToken(ge, false, true))
		err.Element = "GE01"
		doc.Errors = append(doc.Errors, err)
	}
	if group.ControlNumber != "" && value(ge, 2) != "" && group.ControlNumber != value(ge, 2) {
		err := ediError("X12_GROUP_CONTROL_NUMBER_MISMATCH", "GS06 does not match GE02", segmentFromToken(ge, false, true))
		err.Element = "GE02"
		doc.Errors = append(doc.Errors, err)
	}
}

func validateInterchange(doc *model.Document, interchange *model.Interchange, iea Token) {
	if expected, ok := atoi(value(iea, 1)); ok && expected != len(interchange.Groups) {
		err := ediError("X12_INTERCHANGE_COUNT_MISMATCH", fmt.Sprintf("IEA01 declares %d groups but parsed %d", expected, len(interchange.Groups)), segmentFromToken(iea, false, true))
		err.Element = "IEA01"
		doc.Errors = append(doc.Errors, err)
	}
	if interchange.ControlNumber != "" && value(iea, 2) != "" && interchange.ControlNumber != value(iea, 2) {
		err := ediError("X12_INTERCHANGE_CONTROL_NUMBER_MISMATCH", "ISA13 does not match IEA02", segmentFromToken(iea, false, true))
		err.Element = "IEA02"
		doc.Errors = append(doc.Errors, err)
	}
}

func summarize(doc *model.Document) {
	for _, interchange := range doc.Interchanges {
		doc.Metadata.Groups += len(interchange.Groups)
		for _, group := range interchange.Groups {
			doc.Metadata.Transactions += len(group.Transactions)
		}
	}
	doc.Metadata.TranslatedBy = "EDIForge"
}

func value(tok Token, index int) string {
	if index <= 0 || index > len(tok.Elements) {
		return ""
	}
	return strings.TrimSpace(tok.Elements[index-1].Value)
}

func atoi(s string) (int, bool) {
	i, err := strconv.Atoi(strings.TrimSpace(s))
	return i, err == nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func ediError(code, message string, seg model.Segment) model.EDIError {
	return model.EDIError{
		Severity:        "error",
		Code:            code,
		Message:         message,
		Standard:        string(model.StandardX12),
		Segment:         seg.Tag,
		SegmentPosition: seg.Position,
		ByteOffset:      seg.Offset,
	}
}
