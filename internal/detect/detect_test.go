package detect

import (
	"os"
	"testing"

	"github.com/openedi/ediforge/internal/model"
)

func TestDetectX12(t *testing.T) {
	data, err := os.ReadFile("../../testdata/x12/850-basic.edi")
	if err != nil {
		t.Fatal(err)
	}
	result, err := Detect(data, model.StandardAuto)
	if err != nil {
		t.Fatal(err)
	}
	if result.Standard != model.StandardX12 {
		t.Fatalf("standard = %s", result.Standard)
	}
	if result.Delimiters.Element != "*" || result.Delimiters.Segment != "~" || result.Delimiters.Component != ">" {
		t.Fatalf("unexpected delimiters: %+v", result.Delimiters)
	}
}

func TestDetectEDIFACT(t *testing.T) {
	data, err := os.ReadFile("../../testdata/edifact/orders-basic.edi")
	if err != nil {
		t.Fatal(err)
	}
	result, err := Detect(data, model.StandardAuto)
	if err != nil {
		t.Fatal(err)
	}
	if result.Standard != model.StandardEDIFACT {
		t.Fatalf("standard = %s", result.Standard)
	}
	if result.Delimiters.Element != "+" || result.Delimiters.Segment != "'" || result.Delimiters.Component != ":" || result.Delimiters.Release != "?" {
		t.Fatalf("unexpected delimiters: %+v", result.Delimiters)
	}
}
