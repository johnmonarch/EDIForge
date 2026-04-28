package model

type Standard string

const (
	StandardAuto    Standard = "auto"
	StandardX12     Standard = "x12"
	StandardEDIFACT Standard = "edifact"
	StandardUnknown Standard = "unknown"
)

type Mode string

const (
	ModeStructural Mode = "structural"
	ModeAnnotated  Mode = "annotated"
	ModeSemantic   Mode = "semantic"
)

type Delimiters struct {
	Element     string `json:"element,omitempty"`
	Segment     string `json:"segment,omitempty"`
	Component   string `json:"component,omitempty"`
	Repetition  string `json:"repetition,omitempty"`
	Release     string `json:"release,omitempty"`
	DecimalMark string `json:"decimalMark,omitempty"`
}

type Metadata struct {
	InputName     string     `json:"inputName,omitempty"`
	ParseMs       int64      `json:"parseMs,omitempty"`
	Segments      int        `json:"segments"`
	Groups        int        `json:"groups,omitempty"`
	Transactions  int        `json:"transactions,omitempty"`
	Messages      int        `json:"messages,omitempty"`
	Delimiters    Delimiters `json:"delimiters,omitempty"`
	SchemaID      string     `json:"schemaId,omitempty"`
	TranslatedBy  string     `json:"translatedBy,omitempty"`
	EffectiveMode Mode       `json:"mode,omitempty"`
}

type Document struct {
	Standard     Standard      `json:"standard"`
	Version      string        `json:"version,omitempty"`
	Interchanges []Interchange `json:"interchanges"`
	Errors       []EDIError    `json:"errors,omitempty"`
	Warnings     []EDIWarning  `json:"warnings,omitempty"`
	Metadata     Metadata      `json:"metadata"`
}

type Interchange struct {
	Standard      Standard  `json:"standard"`
	SenderID      string    `json:"senderId,omitempty"`
	ReceiverID    string    `json:"receiverId,omitempty"`
	ControlNumber string    `json:"controlNumber,omitempty"`
	Groups        []Group   `json:"groups,omitempty"`
	Messages      []Message `json:"messages,omitempty"`
	RawEnvelope   []Segment `json:"rawEnvelope,omitempty"`
}

type Group struct {
	FunctionalID  string        `json:"functionalId,omitempty"`
	Version       string        `json:"version,omitempty"`
	ControlNumber string        `json:"controlNumber,omitempty"`
	Transactions  []Transaction `json:"transactions"`
}

type Transaction struct {
	Type          string    `json:"type"`
	Version       string    `json:"version,omitempty"`
	ControlNumber string    `json:"controlNumber,omitempty"`
	Segments      []Segment `json:"segments"`
	SegmentCount  int       `json:"segmentCount"`
}

type Message struct {
	Type            string    `json:"type"`
	Version         string    `json:"version,omitempty"`
	Release         string    `json:"release,omitempty"`
	ControllingOrg  string    `json:"controllingOrg,omitempty"`
	AssociationCode string    `json:"associationCode,omitempty"`
	Reference       string    `json:"reference,omitempty"`
	Segments        []Segment `json:"segments"`
	SegmentCount    int       `json:"segmentCount"`
}

type Segment struct {
	Tag      string    `json:"tag"`
	Position int       `json:"position"`
	Elements []Element `json:"elements"`
	Raw      string    `json:"raw,omitempty"`
	Offset   int64     `json:"offset,omitempty"`
}

type Element struct {
	Index      int      `json:"index"`
	Value      string   `json:"value,omitempty"`
	Components []string `json:"components,omitempty"`
}

type EDIError struct {
	Severity        string `json:"severity"`
	Code            string `json:"code"`
	Message         string `json:"message"`
	Standard        string `json:"standard,omitempty"`
	Segment         string `json:"segment,omitempty"`
	SegmentPosition int    `json:"segmentPosition,omitempty"`
	Element         string `json:"element,omitempty"`
	ByteOffset      int64  `json:"byteOffset,omitempty"`
	Hint            string `json:"hint,omitempty"`
}

type EDIWarning struct {
	Severity        string `json:"severity"`
	Code            string `json:"code"`
	Message         string `json:"message"`
	Standard        string `json:"standard,omitempty"`
	Segment         string `json:"segment,omitempty"`
	SegmentPosition int    `json:"segmentPosition,omitempty"`
	Element         string `json:"element,omitempty"`
	ByteOffset      int64  `json:"byteOffset,omitempty"`
	Hint            string `json:"hint,omitempty"`
}

func Error(code, message string, standard Standard, segment Segment) EDIError {
	return EDIError{
		Severity:        "error",
		Code:            code,
		Message:         message,
		Standard:        string(standard),
		Segment:         segment.Tag,
		SegmentPosition: segment.Position,
		ByteOffset:      segment.Offset,
	}
}

func Warning(code, message string, standard Standard, segment Segment) EDIWarning {
	return EDIWarning{
		Severity:        "warning",
		Code:            code,
		Message:         message,
		Standard:        string(standard),
		Segment:         segment.Tag,
		SegmentPosition: segment.Position,
		ByteOffset:      segment.Offset,
	}
}

func ElementValue(elements []Element, index int) string {
	if index <= 0 || index > len(elements) {
		return ""
	}
	return elements[index-1].Value
}

func ElementComponent(elements []Element, index, component int) string {
	if index <= 0 || index > len(elements) {
		return ""
	}
	el := elements[index-1]
	if component <= 0 {
		return el.Value
	}
	if component == 1 && len(el.Components) == 0 {
		return el.Value
	}
	if component > len(el.Components) {
		return ""
	}
	return el.Components[component-1]
}
