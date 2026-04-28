package x12

import (
	"testing"

	"github.com/johnmonarch/ediforge/internal/model"
)

func TestTokenizeX12MalformedInvalidTagsAndOffsets(t *testing.T) {
	input := "A*bad~GOOD*one>two**three~TOOLONG*bad~"
	tokens, errs := Tokenize(input, Options{
		Delimiters:     model.Delimiters{Element: "*", Segment: "~", Component: ">"},
		IncludeRaw:     true,
		IncludeOffsets: true,
	})
	if len(tokens) != 3 {
		t.Fatalf("tokens = %d", len(tokens))
	}
	assertX12TokenizerErrorCodes(t, errs, "X12_INVALID_SEGMENT_TAG")
	if tokens[0].Offset != 0 || tokens[1].Offset != int64(len("A*bad~")) {
		t.Fatalf("offsets = %d, %d", tokens[0].Offset, tokens[1].Offset)
	}
	if got := tokens[1].Elements[0].Components; len(got) != 2 || got[0] != "one" || got[1] != "two" {
		t.Fatalf("components = %#v", got)
	}
	if tokens[1].Elements[1].Value != "" {
		t.Fatalf("empty element value = %q", tokens[1].Elements[1].Value)
	}
}

func FuzzTokenizeX12(f *testing.F) {
	seeds := []string{
		"",
		"ISA*00*          *00*          *ZZ*SENDER         *ZZ*RECEIVER       *260427*1200*U*00401*000000001*0*T*>~",
		"ST*850*0001~BEG*00*SA*PO12345~SE*3*0001~",
		"A*bad~TOOLONG*bad~",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	opts := Options{Delimiters: model.Delimiters{Element: "*", Segment: "~", Component: ">"}, IncludeRaw: true, IncludeOffsets: true}
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

func assertX12TokenizerErrorCodes(t *testing.T, errs []model.EDIError, want ...string) {
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
