package detect

import (
	"os"
	"strings"
	"testing"

	"github.com/johnmonarch/ediforge/internal/model"
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

func TestDetectX12UsesFixedISAPositions(t *testing.T) {
	data := []byte(x12ISA("|", ":", "!", "^") + "GS|PO|SENDER|RECEIVER|20240101|1253|1|X|005010!")
	result, err := Detect(data, model.StandardAuto)
	if err != nil {
		t.Fatal(err)
	}
	if result.Delimiters.Element != "|" || result.Delimiters.Segment != "!" || result.Delimiters.Component != ":" || result.Delimiters.Repetition != "^" {
		t.Fatalf("unexpected delimiters: %+v", result.Delimiters)
	}
	if result.Version != "00501" {
		t.Fatalf("version = %q", result.Version)
	}
}

func TestDetectX12ShortISADoesNotPanic(t *testing.T) {
	result, err := Detect([]byte("ISA|00|short"), model.StandardX12)
	if err != nil {
		t.Fatal(err)
	}
	if result.Delimiters.Element != "|" || result.Delimiters.Segment != "~" || result.Delimiters.Component != ">" {
		t.Fatalf("unexpected delimiters: %+v", result.Delimiters)
	}
}

func TestDetectEDIFACTUsesUNAServiceStringAdvice(t *testing.T) {
	data := []byte("UNA*|.! ~UNB|UNOC*3|SENDER|RECEIVER|240101*1253|1~")
	result, err := Detect(data, model.StandardAuto)
	if err != nil {
		t.Fatal(err)
	}
	if result.Delimiters.Component != "*" || result.Delimiters.Element != "|" || result.Delimiters.DecimalMark != "." || result.Delimiters.Release != "!" || result.Delimiters.Segment != "~" {
		t.Fatalf("unexpected delimiters: %+v", result.Delimiters)
	}
}

func x12ISA(element, component, segment, repetition string) string {
	fields := []string{
		"00",
		"          ",
		"00",
		"          ",
		"ZZ",
		"SENDER         ",
		"ZZ",
		"RECEIVER       ",
		"240101",
		"1253",
		repetition,
		"00501",
		"000000905",
		"0",
		"T",
		component,
	}
	return "ISA" + element + strings.Join(fields, element) + segment
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
