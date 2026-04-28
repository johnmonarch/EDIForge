package edifact

import (
	"context"
	"os"
	"testing"

	"github.com/johnmonarch/ediforge/internal/model"
)

func TestParseEDIFACTEnvelope(t *testing.T) {
	data, err := os.ReadFile("../../../testdata/edifact/orders-basic.edi")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Parse(context.Background(), string(data), Options{Delimiters: model.Delimiters{Element: "+", Segment: "'", Component: ":", Release: "?"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Errors) != 0 {
		t.Fatalf("errors = %+v", doc.Errors)
	}
	if len(doc.Interchanges) != 1 {
		t.Fatalf("interchanges = %d", len(doc.Interchanges))
	}
	msg := doc.Interchanges[0].Messages[0]
	if msg.Type != "ORDERS" || msg.Reference != "1" || msg.SegmentCount != 5 {
		t.Fatalf("message = %+v", msg)
	}
}

func TestTokenizeEDIFACTReleaseCharacter(t *testing.T) {
	tokens, errs := Tokenize("UNH+1+ORDERS:D:96A:UN'FTX+AAI+++Text with ?+ plus sign'UNT+3+1'", Options{Delimiters: model.Delimiters{Element: "+", Segment: "'", Component: ":", Release: "?"}})
	if len(errs) != 0 {
		t.Fatalf("errs = %+v", errs)
	}
	if got := tokens[1].Elements[3].Value; got != "Text with + plus sign" {
		t.Fatalf("released value = %q", got)
	}
}

func TestParseEDIFACTMalformedTrailerMismatches(t *testing.T) {
	data, err := os.ReadFile("../../../testdata/malformed/edifact-mismatched-trailers.edi")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Parse(context.Background(), string(data), Options{Delimiters: model.Delimiters{Element: "+", Segment: "'", Component: ":", Release: "?"}})
	if err != nil {
		t.Fatal(err)
	}

	assertEDIFACTErrorCodes(t, doc.Errors,
		"EDIFACT_MESSAGE_REFERENCE_MISMATCH",
		"EDIFACT_SEGMENT_COUNT_MISMATCH",
		"EDIFACT_INTERCHANGE_COUNT_MISMATCH",
		"EDIFACT_CONTROL_REFERENCE_MISMATCH",
	)
	if len(doc.Interchanges) != 1 || len(doc.Interchanges[0].Messages) != 1 {
		t.Fatalf("recovered envelope = %+v", doc.Interchanges)
	}
	if got := doc.Interchanges[0].Messages[0].SegmentCount; got != 3 {
		t.Fatalf("recovered message segment count = %d", got)
	}
}

func TestParseEDIFACTMalformedMissingUNT(t *testing.T) {
	input := "UNB+UNOC:3+SENDER+RECEIVER+260427:1200+1'UNH+1+ORDERS:D:96A:UN'BGM+220+PO12345+9'UNZ+1+1'"
	doc, err := Parse(context.Background(), input, Options{Delimiters: model.Delimiters{Element: "+", Segment: "'", Component: ":", Release: "?"}})
	if err != nil {
		t.Fatal(err)
	}

	assertEDIFACTErrorCodes(t, doc.Errors, "EDIFACT_MISSING_UNT")
	if len(doc.Interchanges) != 1 || len(doc.Interchanges[0].Messages) != 1 {
		t.Fatalf("recovered envelope = %+v", doc.Interchanges)
	}
	if got := doc.Metadata.Messages; got != 1 {
		t.Fatalf("metadata messages = %d", got)
	}
}

func FuzzParseEDIFACT(f *testing.F) {
	seeds := []string{
		"",
		"UNA:+.? 'UNB+UNOC:3+SENDER+RECEIVER+260427:1200+1'UNH+1+ORDERS:D:96A:UN'BGM+220+PO12345+9'UNT+3+1'UNZ+1+1'",
		"UNH+1+ORDERS:D:96A:UN'BGM+220+PO12345+9'",
		"UNB+UNOC:3+SENDER+RECEIVER+260427:1200+1?+bad",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	opts := Options{Delimiters: model.Delimiters{Element: "+", Segment: "'", Component: ":", Release: "?"}, IncludeRaw: true, IncludeOffsets: true}
	f.Fuzz(func(t *testing.T, input string) {
		doc, err := Parse(context.Background(), input, opts)
		if err != nil {
			t.Fatalf("Parse returned unexpected error: %v", err)
		}
		if doc == nil {
			t.Fatal("Parse returned nil document")
		}
		if doc.Standard != model.StandardEDIFACT {
			t.Fatalf("standard = %q", doc.Standard)
		}
		if doc.Metadata.Segments < 0 {
			t.Fatalf("negative segment count = %d", doc.Metadata.Segments)
		}
	})
}

func assertEDIFACTErrorCodes(t *testing.T, errs []model.EDIError, want ...string) {
	t.Helper()
	got := make(map[string]bool, len(errs))
	for _, err := range errs {
		got[err.Code] = true
	}
	for _, code := range want {
		if !got[code] {
			t.Fatalf("missing error code %s in %+v", code, errs)
		}
	}
}
