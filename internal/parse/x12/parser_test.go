package x12

import (
	"context"
	"os"
	"testing"

	"github.com/johnmonarch/ediforge/internal/model"
)

func TestParseX12Envelope(t *testing.T) {
	data, err := os.ReadFile("../../../testdata/x12/850-basic.edi")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Parse(context.Background(), string(data), Options{Delimiters: model.Delimiters{Element: "*", Segment: "~", Component: ">"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Errors) != 0 {
		t.Fatalf("errors = %+v", doc.Errors)
	}
	if len(doc.Interchanges) != 1 {
		t.Fatalf("interchanges = %d", len(doc.Interchanges))
	}
	group := doc.Interchanges[0].Groups[0]
	if group.FunctionalID != "PO" || len(group.Transactions) != 1 {
		t.Fatalf("group = %+v", group)
	}
	tx := group.Transactions[0]
	if tx.Type != "850" || tx.SegmentCount != 5 {
		t.Fatalf("transaction = %+v", tx)
	}
}

func TestParseX12MalformedTrailerMismatches(t *testing.T) {
	data, err := os.ReadFile("../../../testdata/malformed/x12-mismatched-trailers.edi")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Parse(context.Background(), string(data), Options{Delimiters: model.Delimiters{Element: "*", Segment: "~", Component: ">"}})
	if err != nil {
		t.Fatal(err)
	}

	assertX12ErrorCodes(t, doc.Errors,
		"X12_CONTROL_NUMBER_MISMATCH",
		"X12_SEGMENT_COUNT_MISMATCH",
		"X12_GROUP_COUNT_MISMATCH",
		"X12_GROUP_CONTROL_NUMBER_MISMATCH",
		"X12_INTERCHANGE_COUNT_MISMATCH",
		"X12_INTERCHANGE_CONTROL_NUMBER_MISMATCH",
	)
	if len(doc.Interchanges) != 1 || len(doc.Interchanges[0].Groups) != 1 {
		t.Fatalf("recovered envelope = %+v", doc.Interchanges)
	}
	if got := doc.Interchanges[0].Groups[0].Transactions[0].SegmentCount; got != 3 {
		t.Fatalf("recovered transaction segment count = %d", got)
	}
}

func TestParseX12MalformedUnclosedEnvelope(t *testing.T) {
	input := "ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *260427*1200*U*00401*000000001*0*T*>~GS*PO*SENDER*RECEIVER*20260427*1200*1*X*004010~ST*850*0001~BEG*00*SA*PO12345**20260427~"
	doc, err := Parse(context.Background(), input, Options{Delimiters: model.Delimiters{Element: "*", Segment: "~", Component: ">"}})
	if err != nil {
		t.Fatal(err)
	}

	assertX12ErrorCodes(t, doc.Errors, "X12_MISSING_SE", "X12_MISSING_GE", "X12_MISSING_IEA")
	if len(doc.Interchanges) != 1 {
		t.Fatalf("interchanges = %d", len(doc.Interchanges))
	}
	if got := doc.Metadata.Transactions; got != 1 {
		t.Fatalf("metadata transactions = %d", got)
	}
}

func FuzzParseX12(f *testing.F) {
	seeds := []string{
		"",
		"ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *260427*1200*U*00401*000000001*0*T*>~GS*PO*SENDER*RECEIVER*20260427*1200*1*X*004010~ST*850*0001~BEG*00*SA*PO12345**20260427~SE*4*0001~GE*1*1~IEA*1*000000001~",
		"ST*850*0001~BEG*00*SA*PO12345~",
		"ISA*00*bad~GS*PO~ST*850~SE*2*1~",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	opts := Options{Delimiters: model.Delimiters{Element: "*", Segment: "~", Component: ">"}, IncludeRaw: true, IncludeOffsets: true}
	f.Fuzz(func(t *testing.T, input string) {
		doc, err := Parse(context.Background(), input, opts)
		if err != nil {
			t.Fatalf("Parse returned unexpected error: %v", err)
		}
		if doc == nil {
			t.Fatal("Parse returned nil document")
		}
		if doc.Standard != model.StandardX12 {
			t.Fatalf("standard = %q", doc.Standard)
		}
		if doc.Metadata.Segments < 0 {
			t.Fatalf("negative segment count = %d", doc.Metadata.Segments)
		}
	})
}

func assertX12ErrorCodes(t *testing.T, errs []model.EDIError, want ...string) {
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
