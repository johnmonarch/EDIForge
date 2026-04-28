package translate

import (
	"context"
	"strings"
	"testing"

	"github.com/johnmonarch/ediforge/internal/model"
)

func TestTranslateUsesDetectedX12DelimitersWithReaderParse(t *testing.T) {
	input := x12ISA("|", ":", "!", "^") +
		"GS|PO|SENDER|RECEIVER|20240101|1253|1|X|005010!" +
		"ST|850|0001!" +
		"SE|2|0001!" +
		"GE|1|1!" +
		"IEA|1|000000905!"

	result, err := NewService().Translate(context.Background(), Input{Reader: strings.NewReader(input)}, TranslateOptions{
		Standard: model.StandardAuto,
		Mode:     model.ModeStructural,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("translate was not ok: %+v", result.Errors)
	}
	delims := result.Metadata.Delimiters
	if delims.Element != "|" || delims.Segment != "!" || delims.Component != ":" {
		t.Fatalf("unexpected delimiters: %+v", delims)
	}
}

func TestTranslateUsesDetectedEDIFACTDelimitersWithReaderParse(t *testing.T) {
	input := "UNA*|.! ~" +
		"UNB|UNOC*3|SENDER|RECEIVER|240101*1253|1~" +
		"UNH|1|ORDERS*D*96A*UN~" +
		"UNT|2|1~" +
		"UNZ|1|1~"

	result, err := NewService().Translate(context.Background(), Input{Reader: strings.NewReader(input)}, TranslateOptions{
		Standard: model.StandardAuto,
		Mode:     model.ModeStructural,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("translate was not ok: %+v", result.Errors)
	}
	delims := result.Metadata.Delimiters
	if delims.Component != "*" || delims.Element != "|" || delims.Release != "!" || delims.Segment != "~" {
		t.Fatalf("unexpected delimiters: %+v", delims)
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
