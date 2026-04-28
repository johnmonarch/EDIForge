package x12

import (
	"context"
	"strings"
	"testing"
)

func TestParseReaderDerivesX12DelimitersFromISA(t *testing.T) {
	input := x12ISA("|", ":", "!", "^") +
		"GS|PO|SENDER|RECEIVER|20240101|1253|1|X|005010!" +
		"ST|850|0001!" +
		"SE|2|0001!" +
		"GE|1|1!" +
		"IEA|1|000000905!"

	doc, err := ParseReader(context.Background(), strings.NewReader(input), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Metadata.Delimiters.Element != "|" || doc.Metadata.Delimiters.Segment != "!" || doc.Metadata.Delimiters.Component != ":" {
		t.Fatalf("unexpected delimiters: %+v", doc.Metadata.Delimiters)
	}
	if len(doc.Errors) != 0 {
		t.Fatalf("unexpected parse errors: %+v", doc.Errors)
	}
}

func TestTokenizeReaderMatchesStringAPI(t *testing.T) {
	input := "ST*850*0001~SE*2*0001~"
	fromString, stringErrs := Tokenize(input, Options{})
	fromReader, readerErrs := TokenizeReader(strings.NewReader(input), Options{})
	if len(stringErrs) != len(readerErrs) {
		t.Fatalf("error counts differ: string=%d reader=%d", len(stringErrs), len(readerErrs))
	}
	if len(fromString) != len(fromReader) {
		t.Fatalf("token counts differ: string=%d reader=%d", len(fromString), len(fromReader))
	}
	for i := range fromString {
		if fromString[i].Tag != fromReader[i].Tag || len(fromString[i].Elements) != len(fromReader[i].Elements) {
			t.Fatalf("token %d differs: string=%+v reader=%+v", i, fromString[i], fromReader[i])
		}
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
