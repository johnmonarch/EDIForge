package edifact

import (
	"context"
	"strings"
	"testing"
)

func TestParseReaderDerivesEDIFACTDelimitersFromUNA(t *testing.T) {
	input := "UNA*|.! ~" +
		"UNB|UNOC*3|SENDER|RECEIVER|240101*1253|1~" +
		"UNH|1|ORDERS*D*96A*UN~" +
		"UNT|2|1~" +
		"UNZ|1|1~"

	doc, err := ParseReader(context.Background(), strings.NewReader(input), Options{})
	if err != nil {
		t.Fatal(err)
	}
	delims := doc.Metadata.Delimiters
	if delims.Component != "*" || delims.Element != "|" || delims.Release != "!" || delims.Segment != "~" || delims.DecimalMark != "." {
		t.Fatalf("unexpected delimiters: %+v", delims)
	}
	if len(doc.Errors) != 0 {
		t.Fatalf("unexpected parse errors: %+v", doc.Errors)
	}
}

func TestTokenizeReaderMatchesStringAPI(t *testing.T) {
	input := "UNH+1+ORDERS:D:96A:UN'UNT+2+1'"
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
