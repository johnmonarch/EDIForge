package mapping

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/openedi/ediforge/internal/model"
	"github.com/openedi/ediforge/internal/schema"
)

func Map(doc *model.Document, s *schema.Schema) (map[string]any, []model.EDIWarning, []model.EDIError) {
	output := map[string]any{
		"standard": string(doc.Standard),
	}
	if s.Output.DocumentType != "" {
		output["documentType"] = s.Output.DocumentType
	}
	if s.Transaction != "" {
		output["sourceType"] = s.Transaction
	}
	if s.Message != "" {
		output["sourceType"] = s.Message
	}

	segments := flattenSegments(doc)
	var warnings []model.EDIWarning
	var errors []model.EDIError

	for target, rule := range s.Mapping {
		if rule.Literal != "" {
			setValue(output, target, rule.Literal)
			continue
		}
		if rule.Path == "" {
			continue
		}
		if strings.Contains(target, "[]") {
			if err := mapArrayRule(output, target, rule, segments); err != nil {
				errors = append(errors, mappingError(doc.Standard, target, err))
			}
			continue
		}
		value, required, err := evalRule(rule.Path, rule.Transforms, segments)
		if err != nil {
			errors = append(errors, mappingError(doc.Standard, target, err))
			continue
		}
		if required && value == "" {
			errors = append(errors, requiredError(doc.Standard, target))
			continue
		}
		if value == "" {
			warnings = append(warnings, emptyWarning(doc.Standard, target))
			continue
		}
		setValue(output, target, value)
	}

	for target, expression := range s.Maps {
		if _, exists := s.Mapping[target]; exists {
			continue
		}
		value, required, err := evalExpression(expression, segments)
		if err != nil {
			errors = append(errors, mappingError(doc.Standard, target, err))
			continue
		}
		if required && value == "" {
			errors = append(errors, requiredError(doc.Standard, target))
			continue
		}
		if value == "" {
			warnings = append(warnings, emptyWarning(doc.Standard, target))
			continue
		}
		setValue(output, target, value)
	}
	return output, warnings, errors
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

func mapArrayRule(output map[string]any, target string, rule schema.MappingRule, segments []model.Segment) error {
	arrayName, itemPath, ok := splitArrayTarget(target)
	if !ok {
		return fmt.Errorf("invalid array target %q", target)
	}
	values, required, err := evalArrayRule(rule.Path, rule.Transforms, segments)
	if err != nil {
		return err
	}
	if len(values) == 0 {
		return nil
	}
	items := getArray(output, arrayName)
	for len(items) < len(values) {
		items = append(items, map[string]any{})
	}
	for i, value := range values {
		if required && value == "" {
			return fmt.Errorf("required mapped array field %q[%d] was empty", target, i)
		}
		if value == "" {
			continue
		}
		setNestedValue(items[i], itemPath, value)
	}
	output[arrayName] = items
	return nil
}

func evalRule(source string, transforms []string, segments []model.Segment) (string, bool, error) {
	value, err := resolveSource(source, segments, nil)
	if err != nil {
		return "", false, err
	}
	return applyTransforms(value, transforms)
}

func evalExpression(expression string, segments []model.Segment) (string, bool, error) {
	parts := strings.Split(expression, "|")
	source := strings.TrimSpace(parts[0])
	value, err := resolveSource(source, segments, nil)
	if err != nil {
		return "", false, err
	}
	var transforms []string
	for _, transform := range parts[1:] {
		transforms = append(transforms, strings.TrimSpace(transform))
	}
	return applyTransforms(value, transforms)
}

func evalArrayRule(source string, transforms []string, segments []model.Segment) ([]string, bool, error) {
	chain, err := parsePath(source)
	if err != nil {
		return nil, false, err
	}
	parent := chain[0]
	if !parent.Spec.Selector.All {
		return nil, false, fmt.Errorf("array mapping source %q must start with a [] selector", source)
	}
	parents := selectSegments(segments, parent.Spec)
	values := make([]string, 0, len(parents))
	required := false
	for _, parentRef := range parents {
		var value string
		if len(chain) == 1 {
			value = fieldValue(parentRef.Segment, parent.Field)
		} else {
			scope := childScope(segments, parentRef.Index, parent.Spec.Tag)
			value, err = resolveParsedPath(chain[1:], scope)
			if err != nil {
				return nil, required, err
			}
		}
		var itemRequired bool
		value, itemRequired, err = applyTransforms(value, transforms)
		if err != nil {
			return nil, required, err
		}
		required = required || itemRequired
		values = append(values, value)
	}
	return values, required, nil
}

func resolveSource(source string, segments []model.Segment, context *segmentRef) (string, error) {
	chain, err := parsePath(source)
	if err != nil {
		return "", err
	}
	if context != nil && len(chain) == 1 {
		return fieldValue(context.Segment, chain[0].Field), nil
	}
	return resolveParsedPath(chain, segments)
}

func resolveParsedPath(chain []pathStep, segments []model.Segment) (string, error) {
	if len(chain) == 0 {
		return "", fmt.Errorf("empty source path")
	}
	refs := selectSegments(segments, chain[0].Spec)
	if len(refs) == 0 {
		return "", nil
	}
	ref := refs[0]
	if len(chain) == 1 {
		return fieldValue(ref.Segment, chain[0].Field), nil
	}
	scope := childScope(segments, ref.Index, chain[0].Spec.Tag)
	return resolveParsedPath(chain[1:], scope)
}

func parsePath(source string) ([]pathStep, error) {
	parts := strings.Split(source, ">")
	steps := make([]pathStep, 0, len(parts))
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		segSpec, fieldSpec, ok := splitSource(part)
		if !ok {
			if i != len(parts)-1 {
				spec, err := parseSegmentSpec(part)
				if err != nil {
					return nil, err
				}
				steps = append(steps, pathStep{Spec: spec})
				continue
			}
			return nil, fmt.Errorf("invalid source path %q", source)
		}
		spec, err := parseSegmentSpec(segSpec)
		if err != nil {
			return nil, err
		}
		field, err := parseFieldRef(fieldSpec, spec.Tag)
		if err != nil {
			return nil, err
		}
		steps = append(steps, pathStep{Spec: spec, Field: field})
	}
	if len(steps) == 0 {
		return nil, fmt.Errorf("invalid source path %q", source)
	}
	return steps, nil
}

func splitSource(source string) (string, string, bool) {
	source = strings.TrimSpace(source)
	if close := strings.Index(source, "]."); close >= 0 {
		return source[:close+1], source[close+2:], true
	}
	idx := strings.Index(source, ".")
	if idx < 0 {
		return "", "", false
	}
	return source[:idx], source[idx+1:], true
}

type pathStep struct {
	Spec  segmentSpec
	Field fieldRef
}

type segmentSpec struct {
	Tag      string
	Selector selector
}

type selector struct {
	Occurrence int
	All        bool
	Filter     *filter
}

type filter struct {
	Field fieldRef
	Value string
}

type fieldRef struct {
	Index     int
	Component int
}

type segmentRef struct {
	Index   int
	Segment model.Segment
}

func parseSegmentSpec(spec string) (segmentSpec, error) {
	if idx := strings.Index(spec, "["); idx >= 0 {
		if !strings.HasSuffix(spec, "]") {
			return segmentSpec{}, fmt.Errorf("invalid segment selector %q", spec)
		}
		tag := strings.TrimSpace(spec[:idx])
		body := strings.TrimSpace(strings.TrimSuffix(spec[idx+1:], "]"))
		switch {
		case body == "":
			return segmentSpec{Tag: tag, Selector: selector{All: true}}, nil
		case body == "*":
			return segmentSpec{Tag: tag, Selector: selector{All: true}}, nil
		}
		if n, err := strconv.Atoi(body); err == nil {
			return segmentSpec{Tag: tag, Selector: selector{Occurrence: n}}, nil
		}
		lhs, rhs, ok := strings.Cut(body, "=")
		if !ok {
			return segmentSpec{}, fmt.Errorf("invalid segment filter %q", body)
		}
		field, err := parseFieldRef(strings.TrimSpace(lhs), tag)
		if err != nil {
			return segmentSpec{}, err
		}
		return segmentSpec{
			Tag: tag,
			Selector: selector{
				Filter: &filter{Field: field, Value: strings.Trim(strings.TrimSpace(rhs), "'\"")},
			},
		}, nil
	}
	return segmentSpec{Tag: strings.TrimSpace(spec), Selector: selector{}}, nil
}

func parseFieldRef(ref, defaultTag string) (fieldRef, error) {
	ref = strings.TrimSpace(ref)
	ref = strings.TrimPrefix(ref, defaultTag)
	parts := strings.SplitN(ref, ".", 2)
	field, err := strconv.Atoi(parts[0])
	if err != nil {
		return fieldRef{}, fmt.Errorf("invalid element reference %q", ref)
	}
	out := fieldRef{Index: field}
	if len(parts) == 2 {
		component, err := strconv.Atoi(parts[1])
		if err != nil {
			return fieldRef{}, fmt.Errorf("invalid component reference %q", ref)
		}
		out.Component = component
	}
	return out, nil
}

func selectSegments(segments []model.Segment, spec segmentSpec) []segmentRef {
	var refs []segmentRef
	occurrence := 0
	for i, segment := range segments {
		if segment.Tag != spec.Tag {
			continue
		}
		if spec.Selector.Filter != nil {
			if fieldValue(segment, spec.Selector.Filter.Field) == spec.Selector.Filter.Value {
				refs = append(refs, segmentRef{Index: i, Segment: segment})
				if !spec.Selector.All {
					return refs
				}
			}
			continue
		}
		if spec.Selector.All {
			refs = append(refs, segmentRef{Index: i, Segment: segment})
			continue
		}
		if occurrence == spec.Selector.Occurrence {
			return []segmentRef{{Index: i, Segment: segment}}
		}
		occurrence++
	}
	return refs
}

func childScope(segments []model.Segment, parentIndex int, parentTag string) []model.Segment {
	if parentIndex < 0 || parentIndex >= len(segments)-1 {
		return nil
	}
	scope := make([]model.Segment, 0)
	for _, segment := range segments[parentIndex+1:] {
		if segment.Tag == parentTag {
			break
		}
		scope = append(scope, segment)
	}
	return scope
}

func fieldValue(segment model.Segment, field fieldRef) string {
	return model.ElementComponent(segment.Elements, field.Index, field.Component)
}

func applyTransforms(value string, transforms []string) (string, bool, error) {
	required := false
	for _, transform := range transforms {
		name := normalizeTransform(transform)
		switch {
		case name == "":
		case name == "required" || name == "required()":
			required = true
		case name == "trim" || name == "trim()":
			value = strings.TrimSpace(value)
		case name == "upper" || name == "upper()":
			value = strings.ToUpper(value)
		case name == "lower" || name == "lower()":
			value = strings.ToLower(value)
		case strings.HasPrefix(name, "date("):
			format := strings.Trim(name[len("date("):], ")")
			format = strings.Trim(format, "'\"")
			converted, err := convertDate(value, format)
			if err != nil {
				return "", required, err
			}
			value = converted
		case strings.HasPrefix(name, "default("):
			if value == "" {
				value = strings.Trim(strings.TrimSuffix(strings.TrimPrefix(name, "default("), ")"), "'\"")
			}
		case name == "decimal" || name == "decimal()" || name == "number" || name == "number()":
			if value != "" {
				parsed, err := strconv.ParseFloat(value, 64)
				if err != nil {
					return "", required, err
				}
				value = strconv.FormatFloat(parsed, 'f', -1, 64)
			}
		case name == "integer" || name == "integer()":
			if value != "" {
				parsed, err := strconv.Atoi(value)
				if err != nil {
					return "", required, err
				}
				value = strconv.Itoa(parsed)
			}
		case name == "string" || name == "string()":
		default:
			return "", required, fmt.Errorf("unsupported transform %q", name)
		}
	}
	return value, required, nil
}

func normalizeTransform(transform string) string {
	transform = strings.TrimSpace(transform)
	if strings.HasPrefix(transform, "date:") {
		return "date('" + strings.TrimPrefix(transform, "date:") + "')"
	}
	return transform
}

func convertDate(value, format string) (string, error) {
	if value == "" {
		return "", nil
	}
	layouts := map[string]string{
		"yyyyMMdd": "20060102",
		"yyMMdd":   "060102",
	}
	layout, ok := layouts[format]
	if !ok {
		return "", fmt.Errorf("unsupported date format %q", format)
	}
	parsed, err := time.Parse(layout, value)
	if err != nil {
		return "", err
	}
	return parsed.Format("2006-01-02"), nil
}

func splitArrayTarget(target string) (string, string, bool) {
	idx := strings.Index(target, "[]")
	if idx < 0 {
		return "", "", false
	}
	arrayName := target[:idx]
	itemPath := strings.TrimPrefix(target[idx+2:], ".")
	if arrayName == "" || itemPath == "" || strings.Contains(itemPath, "[]") {
		return "", "", false
	}
	return arrayName, itemPath, true
}

func getArray(output map[string]any, name string) []map[string]any {
	existing, ok := output[name].([]map[string]any)
	if ok {
		return existing
	}
	if generic, ok := output[name].([]any); ok {
		items := make([]map[string]any, 0, len(generic))
		for _, value := range generic {
			item, ok := value.(map[string]any)
			if !ok {
				item = map[string]any{}
			}
			items = append(items, item)
		}
		return items
	}
	return nil
}

func setValue(output map[string]any, path, value string) {
	if strings.Contains(path, "[]") {
		return
	}
	setNestedValue(output, path, value)
}

func setNestedValue(output map[string]any, path, value string) {
	parts := strings.Split(path, ".")
	current := output
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]any)
		if !ok {
			next = map[string]any{}
			current[part] = next
		}
		current = next
	}
	current[parts[len(parts)-1]] = value
}

func mappingError(standard model.Standard, target string, err error) model.EDIError {
	return model.EDIError{
		Severity: "error",
		Code:     "MAPPING_EXPRESSION_FAILED",
		Message:  fmt.Sprintf("%s: %v", target, err),
		Standard: string(standard),
	}
}

func requiredError(standard model.Standard, target string) model.EDIError {
	return model.EDIError{
		Severity: "error",
		Code:     "MAPPING_REQUIRED_FIELD_MISSING",
		Message:  fmt.Sprintf("required mapped field %q was empty", target),
		Standard: string(standard),
	}
}

func emptyWarning(standard model.Standard, target string) model.EDIWarning {
	return model.EDIWarning{
		Severity: "warning",
		Code:     "MAPPING_EMPTY_FIELD",
		Message:  fmt.Sprintf("mapped field %q was empty", target),
		Standard: string(standard),
	}
}
