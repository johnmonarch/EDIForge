package edifact

import (
	"os"
	"testing"

	"github.com/johnmonarch/ediforge/internal/model"
)

func TestTokenizeEDIFACTMalformedTerminatorsAndTags(t *testing.T) {
	data, err := os.ReadFile("../../../testdata/malformed/edifact-missing-terminator.edi")
	if err != nil {
		t.Fatal(err)
	}
	tokens, errs := Tokenize(string(data), Options{
		Delimiters:     model.Delimiters{Element: "+", Segment: "'", Component: ":", Release: "?"},
		IncludeRaw:     true,
		IncludeOffsets: true,
	})
	if len(tokens) != 2 {
		t.Fatalf("tokens = %d", len(tokens))
	}
	assertEDIFACTTokenizerErrorCodes(t, errs, "EDIFACT_MISSING_SEGMENT_TERMINATOR")
	if tokens[1].Tag != "BGM" {
		t.Fatalf("last token tag = %q", tokens[1].Tag)
	}

	tokens, errs = Tokenize("unh+1'", Options{Delimiters: model.Delimiters{Element: "+", Segment: "'", Component: ":", Release: "?"}})
	if len(tokens) != 1 {
		t.Fatalf("invalid-tag tokens = %d", len(tokens))
	}
	assertEDIFACTTokenizerErrorCodes(t, errs, "EDIFACT_INVALID_SEGMENT_TAG")
}

func TestTokenizeEDIFACTMalformedReleaseAtEOF(t *testing.T) {
	tokens, errs := Tokenize("UNH+1+ORDERS:D:96A:UN'FTX+AAI+++dangling?", Options{Delimiters: model.Delimiters{Element: "+", Segment: "'", Component: ":", Release: "?"}})
	assertEDIFACTTokenizerErrorCodes(t, errs, "EDIFACT_RELEASE_AT_EOF")
	if len(tokens) != 2 {
		t.Fatalf("tokens after dangling release = %d", len(tokens))
	}
	if tokens[1].Tag != "FTX" {
		t.Fatalf("recovered dangling-release token = %+v", tokens[1])
	}
}

func FuzzTokenizeEDIFACT(f *testing.F) {
	seeds := []string{
		"",
		"UNA:+.? 'UNB+UNOC:3+SENDER+RECEIVER+260427:1200+1'UNH+1+ORDERS:D:96A:UN'UNT+2+1'UNZ+1+1'",
		"UNH+1+ORDERS:D:96A:UN'FTX+AAI+++Text with ?+ plus'",
		"unh+1",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	opts := Options{Delimiters: model.Delimiters{Element: "+", Segment: "'", Component: ":", Release: "?"}, IncludeRaw: true, IncludeOffsets: true}
	f.Fuzz(func(t *testing.T, input string) {
		tokens, _ := Tokenize(input, opts)
		for i, tok := range tokens {
			if tok.Position != i+1 {
				t.Fatalf("token %d position = %d", i, tok.Position)
			}
			if tok.Offset < 0 || tok.Offset > int64(len(input)) {
				t.Fatalf("token %d offset %d out of range for input length %d", i, tok.Offset, len(input))
			}
			for j, el := range tok.Elements {
				if el.Index != j+1 {
					t.Fatalf("token %d element %d index = %d", i, j, el.Index)
				}
			}
		}
	})
}

func assertEDIFACTTokenizerErrorCodes(t *testing.T, errs []model.EDIError, want ...string) {
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
