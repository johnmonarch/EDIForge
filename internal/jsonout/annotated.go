package jsonout

import (
	"fmt"
	"strings"

	"github.com/johnmonarch/ediforge/internal/model"
	"github.com/johnmonarch/ediforge/internal/schema"
)

type annotatedDocument struct {
	Standard     model.Standard         `json:"standard"`
	Version      string                 `json:"version,omitempty"`
	Interchanges []annotatedInterchange `json:"interchanges"`
	Errors       []model.EDIError       `json:"errors,omitempty"`
	Warnings     []model.EDIWarning     `json:"warnings,omitempty"`
	Metadata     model.Metadata         `json:"metadata"`
}

type annotatedInterchange struct {
	Standard      model.Standard     `json:"standard"`
	SenderID      string             `json:"senderId,omitempty"`
	ReceiverID    string             `json:"receiverId,omitempty"`
	ControlNumber string             `json:"controlNumber,omitempty"`
	Groups        []annotatedGroup   `json:"groups,omitempty"`
	Messages      []annotatedMessage `json:"messages,omitempty"`
	RawEnvelope   []annotatedSegment `json:"rawEnvelope,omitempty"`
}

type annotatedGroup struct {
	FunctionalID  string                 `json:"functionalId,omitempty"`
	Version       string                 `json:"version,omitempty"`
	ControlNumber string                 `json:"controlNumber,omitempty"`
	Transactions  []annotatedTransaction `json:"transactions"`
}

type annotatedTransaction struct {
	Type          string             `json:"type"`
	Version       string             `json:"version,omitempty"`
	ControlNumber string             `json:"controlNumber,omitempty"`
	Segments      []annotatedSegment `json:"segments"`
	SegmentCount  int                `json:"segmentCount"`
}

type annotatedMessage struct {
	Type            string             `json:"type"`
	Version         string             `json:"version,omitempty"`
	Release         string             `json:"release,omitempty"`
	ControllingOrg  string             `json:"controllingOrg,omitempty"`
	AssociationCode string             `json:"associationCode,omitempty"`
	Reference       string             `json:"reference,omitempty"`
	Segments        []annotatedSegment `json:"segments"`
	SegmentCount    int                `json:"segmentCount"`
}

type annotatedSegment struct {
	Tag      string             `json:"tag"`
	Name     string             `json:"name,omitempty"`
	Purpose  string             `json:"purpose,omitempty"`
	Position int                `json:"position"`
	Loop     string             `json:"loop,omitempty"`
	Required bool               `json:"required,omitempty"`
	Max      int                `json:"max,omitempty"`
	Maps     map[string]string  `json:"maps,omitempty"`
	Elements []annotatedElement `json:"elements"`
	Raw      string             `json:"raw,omitempty"`
	Offset   int64              `json:"offset,omitempty"`
}

type annotatedElement struct {
	Index            int               `json:"index"`
	ID               string            `json:"id,omitempty"`
	Name             string            `json:"name,omitempty"`
	Target           string            `json:"target,omitempty"`
	ComponentTargets map[string]string `json:"componentTargets,omitempty"`
	Value            string            `json:"value,omitempty"`
	Components       []string          `json:"components,omitempty"`
}

type annotator struct {
	segmentRules map[string]schema.SegmentRule
}

func Annotated(doc *model.Document, s *schema.Schema) any {
	if doc == nil {
		return nil
	}
	a := newAnnotator(s)
	out := annotatedDocument{
		Standard: doc.Standard,
		Version:  doc.Version,
		Errors:   doc.Errors,
		Warnings: doc.Warnings,
		Metadata: doc.Metadata,
	}
	out.Interchanges = make([]annotatedInterchange, 0, len(doc.Interchanges))
	for _, interchange := range doc.Interchanges {
		out.Interchanges = append(out.Interchanges, a.interchange(interchange))
	}
	return out
}

func newAnnotator(s *schema.Schema) annotator {
	a := annotator{segmentRules: map[string]schema.SegmentRule{}}
	if s == nil {
		return a
	}
	for _, rule := range s.Segments {
		if rule.Tag == "" {
			continue
		}
		tag := strings.ToUpper(rule.Tag)
		if _, exists := a.segmentRules[tag]; exists {
			continue
		}
		a.segmentRules[tag] = rule
	}
	return a
}

func (a annotator) interchange(in model.Interchange) annotatedInterchange {
	out := annotatedInterchange{
		Standard:      in.Standard,
		SenderID:      in.SenderID,
		ReceiverID:    in.ReceiverID,
		ControlNumber: in.ControlNumber,
	}
	out.Groups = make([]annotatedGroup, 0, len(in.Groups))
	for _, group := range in.Groups {
		out.Groups = append(out.Groups, a.group(group))
	}
	out.Messages = make([]annotatedMessage, 0, len(in.Messages))
	for _, message := range in.Messages {
		out.Messages = append(out.Messages, a.message(message))
	}
	out.RawEnvelope = a.segments(in.RawEnvelope)
	return out
}

func (a annotator) group(in model.Group) annotatedGroup {
	out := annotatedGroup{
		FunctionalID:  in.FunctionalID,
		Version:       in.Version,
		ControlNumber: in.ControlNumber,
	}
	out.Transactions = make([]annotatedTransaction, 0, len(in.Transactions))
	for _, tx := range in.Transactions {
		out.Transactions = append(out.Transactions, a.transaction(tx))
	}
	return out
}

func (a annotator) transaction(in model.Transaction) annotatedTransaction {
	return annotatedTransaction{
		Type:          in.Type,
		Version:       in.Version,
		ControlNumber: in.ControlNumber,
		Segments:      a.segments(in.Segments),
		SegmentCount:  in.SegmentCount,
	}
}

func (a annotator) message(in model.Message) annotatedMessage {
	return annotatedMessage{
		Type:            in.Type,
		Version:         in.Version,
		Release:         in.Release,
		ControllingOrg:  in.ControllingOrg,
		AssociationCode: in.AssociationCode,
		Reference:       in.Reference,
		Segments:        a.segments(in.Segments),
		SegmentCount:    in.SegmentCount,
	}
}

func (a annotator) segments(in []model.Segment) []annotatedSegment {
	if len(in) == 0 {
		return nil
	}
	out := make([]annotatedSegment, 0, len(in))
	for _, segment := range in {
		out = append(out, a.segment(segment))
	}
	return out
}

func (a annotator) segment(in model.Segment) annotatedSegment {
	var rule *schema.SegmentRule
	if found, ok := a.segmentRules[strings.ToUpper(in.Tag)]; ok {
		rule = &found
	}
	out := annotatedSegment{
		Tag:      in.Tag,
		Position: in.Position,
		Raw:      in.Raw,
		Offset:   in.Offset,
	}
	if rule != nil {
		out.Purpose = rule.Purpose
		out.Name = humanizeIdentifier(rule.Purpose)
		out.Loop = rule.Loop
		out.Required = rule.Required
		out.Max = rule.Max
		out.Maps = cloneStringMap(rule.Maps)
	}
	out.Elements = make([]annotatedElement, 0, len(in.Elements))
	for _, element := range in.Elements {
		out.Elements = append(out.Elements, annotateElement(in.Tag, element, rule))
	}
	return out
}

func annotateElement(tag string, in model.Element, rule *schema.SegmentRule) annotatedElement {
	id := elementID(tag, in.Index)
	out := annotatedElement{
		Index:      in.Index,
		ID:         id,
		Value:      in.Value,
		Components: in.Components,
	}
	if rule == nil || id == "" {
		return out
	}
	if target := rule.Maps[id]; target != "" {
		out.Target = target
		out.Name = humanizeTarget(target)
	}
	out.ComponentTargets = componentTargets(rule.Maps, id)
	return out
}

func elementID(tag string, index int) string {
	if tag == "" || index <= 0 {
		return ""
	}
	return fmt.Sprintf("%s%02d", strings.ToUpper(tag), index)
}

func componentTargets(maps map[string]string, id string) map[string]string {
	if len(maps) == 0 || id == "" {
		return nil
	}
	prefix := id + "."
	targets := map[string]string{}
	for source, target := range maps {
		if strings.HasPrefix(source, prefix) {
			targets[source] = target
		}
	}
	if len(targets) == 0 {
		return nil
	}
	return targets
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func humanizeTarget(target string) string {
	target = strings.ReplaceAll(target, "[]", "")
	if idx := strings.LastIndex(target, "."); idx >= 0 {
		target = target[idx+1:]
	}
	return humanizeIdentifier(target)
}

func humanizeIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.NewReplacer("_", " ", "-", " ").Replace(value)

	var b strings.Builder
	var prev rune
	for i, r := range value {
		if i > 0 && shouldInsertSpace(prev, r) {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
		prev = r
	}

	words := strings.Fields(b.String())
	for i, word := range words {
		words[i] = capitalize(word)
	}
	return strings.Join(words, " ")
}

func shouldInsertSpace(prev, current rune) bool {
	return (isLower(prev) && isUpper(current)) ||
		(isLetter(prev) && isDigit(current)) ||
		(isDigit(prev) && isLetter(current))
}

func capitalize(value string) string {
	if value == "" {
		return ""
	}
	runes := []rune(value)
	runes[0] = toUpper(runes[0])
	return string(runes)
}

func isLower(r rune) bool {
	return r >= 'a' && r <= 'z'
}

func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

func isLetter(r rune) bool {
	return isLower(r) || isUpper(r)
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func toUpper(r rune) rune {
	if isLower(r) {
		return r - ('a' - 'A')
	}
	return r
}
