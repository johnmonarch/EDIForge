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
