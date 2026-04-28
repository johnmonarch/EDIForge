package edifact

import (
	"context"
	"os"
	"testing"

	"github.com/openedi/ediforge/internal/model"
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
