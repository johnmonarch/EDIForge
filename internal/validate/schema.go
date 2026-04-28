package validate

import (
	"fmt"

	"github.com/johnmonarch/ediforge/internal/model"
	"github.com/johnmonarch/ediforge/internal/schema"
)

func Schema(doc *model.Document, s *schema.Schema) ([]model.EDIWarning, []model.EDIError) {
	result := SchemaResult(doc, s)
	return result.WarningsAndErrors()
}

func SchemaResult(doc *model.Document, s *schema.Schema) Result {
	var result Result
	if doc == nil || s == nil {
		return result
	}
	if s.Standard != "" && doc.Standard != "" && s.Standard != doc.Standard {
		result.Add(Issue{
			Rule:     Rule{Code: "SCHEMA_STANDARD_MISMATCH", Severity: SeverityError, Path: "$.standard"},
			Message:  fmt.Sprintf("schema standard %q does not match document standard %q", s.Standard, doc.Standard),
			Standard: doc.Standard,
		})
	}
	if s.Transaction != "" && !hasTransaction(doc, s.Transaction) {
		result.Add(Issue{
			Rule:     Rule{Code: "SCHEMA_TRANSACTION_MISMATCH", Severity: SeverityError, Path: "$.interchanges[].groups[].transactions[].type"},
			Message:  fmt.Sprintf("schema expects X12 transaction %q", s.Transaction),
			Standard: doc.Standard,
		})
	}
	if s.Message != "" && !hasMessage(doc, s.Message) {
		result.Add(Issue{
			Rule:     Rule{Code: "SCHEMA_MESSAGE_MISMATCH", Severity: SeverityError, Path: "$.interchanges[].messages[].type"},
			Message:  fmt.Sprintf("schema expects EDIFACT message %q", s.Message),
			Standard: doc.Standard,
		})
	}
	for _, set := range validationSets(doc) {
		validateSegmentRules(&result, set, s.Segments, doc.Standard)
	}
	if len(s.Segments) == 0 {
		result.Add(Issue{
			Rule:     Rule{Code: "SCHEMA_HAS_NO_SEGMENT_RULES", Severity: SeverityWarning, Path: "$.schema.segments"},
			Message:  fmt.Sprintf("schema %q has no segment rules", s.ID),
			Standard: doc.Standard,
		})
	}
	return result
}

type segmentSet struct {
	Path     string
	Kind     string
	Segments []model.Segment
}

func validateSegmentRules(result *Result, set segmentSet, rules []schema.SegmentRule, standard model.Standard) {
	counts := countSegments(set.Segments)
	loopIndexes := loopStartIndexes(rules)
	for ruleIndex, rule := range rules {
		if rule.Tag == "" {
			continue
		}
		if rule.Loop != "" && !isLoopStart(ruleIndex, rule, rules, loopIndexes) {
			anchorIndex, ok := loopIndexes[rule.Loop]
			if !ok {
				continue
			}
			validateLoopChild(result, set, rule, rules[anchorIndex], standard)
			continue
		}
		count := counts[rule.Tag]
		path := fmt.Sprintf("%s.segments[%s]", set.Path, rule.Tag)
		addMissingMinimum(result, rule, count, path, standard)
		if rule.Max > 0 && count > rule.Max {
			result.Add(Issue{
				Rule:     Rule{Code: "SCHEMA_SEGMENT_REPEAT_EXCEEDED", Severity: SeverityError, Path: path},
				Message:  fmt.Sprintf("segment %s appears %d times in %s, max is %d", rule.Tag, count, set.Kind, rule.Max),
				Standard: standard,
				Segment:  rule.Tag,
			})
		}
	}
}

func flattenSegments(doc *model.Document) []model.Segment {
	var segments []model.Segment
	for _, interchange := range doc.Interchanges {
		for _, group := range interchange.Groups {
			for _, tx := range group.Transactions {
				segments = append(segments, tx.Segments...)
			}
		}
		for _, msg := range interchange.Messages {
			segments = append(segments, msg.Segments...)
		}
	}
	return segments
}

func validationSets(doc *model.Document) []segmentSet {
	var sets []segmentSet
	for i, interchange := range doc.Interchanges {
		for j, group := range interchange.Groups {
			for k, tx := range group.Transactions {
				sets = append(sets, segmentSet{
					Path:     fmt.Sprintf("$.interchanges[%d].groups[%d].transactions[%d]", i, j, k),
					Kind:     "transaction",
					Segments: tx.Segments,
				})
			}
		}
		for j, msg := range interchange.Messages {
			sets = append(sets, segmentSet{
				Path:     fmt.Sprintf("$.interchanges[%d].messages[%d]", i, j),
				Kind:     "message",
				Segments: msg.Segments,
			})
		}
	}
	if len(sets) == 0 {
		sets = append(sets, segmentSet{Path: "$", Kind: "document", Segments: flattenSegments(doc)})
	}
	return sets
}

func countSegments(segments []model.Segment) map[string]int {
	counts := map[string]int{}
	for _, segment := range segments {
		counts[segment.Tag]++
	}
	return counts
}

func loopStartIndexes(rules []schema.SegmentRule) map[string]int {
	indexes := map[string]int{}
	for i, rule := range rules {
		if rule.Loop == "" {
			continue
		}
		if _, exists := indexes[rule.Loop]; !exists {
			indexes[rule.Loop] = i
		}
	}
	return indexes
}

func isLoopStart(index int, rule schema.SegmentRule, rules []schema.SegmentRule, starts map[string]int) bool {
	if rule.Loop == "" {
		return false
	}
	return starts[rule.Loop] == index
}

func validateLoopChild(result *Result, set segmentSet, child schema.SegmentRule, anchor schema.SegmentRule, standard model.Standard) {
	ranges := loopRanges(set.Segments, anchor.Tag)
	for loopIndex, segmentRange := range ranges {
		count := 0
		for _, segment := range set.Segments[segmentRange.start:segmentRange.end] {
			if segment.Tag == child.Tag {
				count++
			}
		}
		path := fmt.Sprintf("%s.loops[%s][%d].segments[%s]", set.Path, child.Loop, loopIndex, child.Tag)
		addMissingMinimum(result, child, count, path, standard)
		if child.Max > 0 && count > child.Max {
			result.Add(Issue{
				Rule:     Rule{Code: "SCHEMA_LOOP_SEGMENT_REPEAT_EXCEEDED", Severity: SeverityError, Path: path},
				Message:  fmt.Sprintf("segment %s appears %d times in loop %s, max is %d", child.Tag, count, child.Loop, child.Max),
				Standard: standard,
				Segment:  child.Tag,
			})
		}
	}
}

type segmentRange struct {
	start int
	end   int
}

func loopRanges(segments []model.Segment, anchorTag string) []segmentRange {
	var ranges []segmentRange
	for i, segment := range segments {
		if segment.Tag != anchorTag {
			continue
		}
		end := len(segments)
		for j := i + 1; j < len(segments); j++ {
			if segments[j].Tag == anchorTag {
				end = j
				break
			}
		}
		ranges = append(ranges, segmentRange{start: i, end: end})
	}
	return ranges
}

func addMissingMinimum(result *Result, rule schema.SegmentRule, count int, path string, standard model.Standard) {
	min := rule.Min
	if rule.Required && min == 0 {
		min = 1
	}
	if min == 0 || count >= min {
		return
	}
	code := "SCHEMA_SEGMENT_MINIMUM_NOT_MET"
	message := fmt.Sprintf("segment %s appears %d times, min is %d", rule.Tag, count, min)
	if rule.Required && min == 1 {
		code = "SCHEMA_REQUIRED_SEGMENT_MISSING"
		message = fmt.Sprintf("required segment %s is missing", rule.Tag)
	}
	result.Add(Issue{
		Rule:     Rule{Code: code, Severity: SeverityError, Path: path},
		Message:  message,
		Standard: standard,
		Segment:  rule.Tag,
	})
}

func hasTransaction(doc *model.Document, expected string) bool {
	for _, interchange := range doc.Interchanges {
		for _, group := range interchange.Groups {
			for _, tx := range group.Transactions {
				if tx.Type == expected {
					return true
				}
			}
		}
	}
	return false
}

func hasMessage(doc *model.Document, expected string) bool {
	for _, interchange := range doc.Interchanges {
		for _, msg := range interchange.Messages {
			if msg.Type == expected {
				return true
			}
		}
	}
	return false
}
