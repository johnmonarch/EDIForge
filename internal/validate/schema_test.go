package validate

import (
	"testing"

	"github.com/johnmonarch/ediforge/internal/model"
	"github.com/johnmonarch/ediforge/internal/schema"
)

func TestSchemaLoopScopedRepeatValidation(t *testing.T) {
	doc := &model.Document{
		Standard: model.StandardX12,
		Interchanges: []model.Interchange{{
			Groups: []model.Group{{
				Transactions: []model.Transaction{{
					Type: "850",
					Segments: []model.Segment{
						{Tag: "ST"},
						{Tag: "N1"},
						{Tag: "N3"},
						{Tag: "N4"},
						{Tag: "N1"},
						{Tag: "N3"},
						{Tag: "N4"},
						{Tag: "SE"},
					},
				}},
			}},
		}},
	}
	s := &schema.Schema{
		ID:          "test",
		Standard:    model.StandardX12,
		Transaction: "850",
		Segments: []schema.SegmentRule{
			{Tag: "N1", Loop: "parties", Max: 20},
			{Tag: "N3", Loop: "parties", Max: 1},
			{Tag: "N4", Loop: "parties", Max: 1},
		},
	}

	_, errors := Schema(doc, s)
	if len(errors) != 0 {
		t.Fatalf("errors = %+v, want none", errors)
	}
}

func TestSchemaLoopScopedRepeatExceeded(t *testing.T) {
	doc := &model.Document{
		Standard: model.StandardX12,
		Interchanges: []model.Interchange{{
			Groups: []model.Group{{
				Transactions: []model.Transaction{{
					Type: "850",
					Segments: []model.Segment{
						{Tag: "ST"},
						{Tag: "N1"},
						{Tag: "N4"},
						{Tag: "N4"},
						{Tag: "SE"},
					},
				}},
			}},
		}},
	}
	s := &schema.Schema{
		ID:          "test",
		Standard:    model.StandardX12,
		Transaction: "850",
		Segments: []schema.SegmentRule{
			{Tag: "N1", Loop: "parties", Max: 20},
			{Tag: "N4", Loop: "parties", Max: 1},
		},
	}

	_, errors := Schema(doc, s)
	if !hasEDIError(errors, "SCHEMA_LOOP_SEGMENT_REPEAT_EXCEEDED") {
		t.Fatalf("errors = %+v, want SCHEMA_LOOP_SEGMENT_REPEAT_EXCEEDED", errors)
	}
	if errors[0].Hint == "" {
		t.Fatalf("error hint empty, want validation path")
	}
}

func TestEnvelopeValidatesControlCountsFromParsedDocument(t *testing.T) {
	doc := &model.Document{
		Standard: model.StandardX12,
		Interchanges: []model.Interchange{{
			ControlNumber: "0001",
			RawEnvelope: []model.Segment{
				{Tag: "IEA", Elements: []model.Element{{Index: 1, Value: "2"}, {Index: 2, Value: "0001"}}},
			},
			Groups: []model.Group{{}},
		}},
	}

	_, errors := Envelope(doc)
	if !hasEDIError(errors, "X12_INTERCHANGE_COUNT_MISMATCH") {
		t.Fatalf("errors = %+v, want X12_INTERCHANGE_COUNT_MISMATCH", errors)
	}
}

func hasEDIError(errors []model.EDIError, code string) bool {
	for _, err := range errors {
		if err.Code == code {
			return true
		}
	}
	return false
}
