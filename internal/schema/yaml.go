package schema

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/johnmonarch/ediforge/internal/model"
)

func parseSimpleYAML(input string) (*Schema, error) {
	s := &Schema{Maps: map[string]string{}}
	section := ""
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent == 0 && strings.HasSuffix(trimmed, ":") {
			section = strings.TrimSuffix(trimmed, ":")
			continue
		}
		key, value, ok := splitYAMLKV(trimmed)
		if !ok {
			continue
		}
		value = unquote(value)
		if indent == 0 {
			section = ""
			switch key {
			case "id":
				s.ID = value
			case "standard":
				s.Standard = model.Standard(value)
			case "transaction":
				s.Transaction = value
			case "message":
				s.Message = value
			case "version":
				s.Version = value
			case "name":
				s.Name = value
			case "license":
				s.License = value
			case "source":
				s.Source = value
			case "documentType":
				s.DocumentType = value
			}
			continue
		}
		switch section {
		case "maps":
			s.Maps[key] = value
		case "output":
			if key == "documentType" {
				s.Output.DocumentType = value
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if s.Standard == "" {
		return nil, fmt.Errorf("schema YAML did not define standard")
	}
	return s, nil
}

func splitYAMLKV(line string) (string, string, bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])
	return key, value, key != ""
}

func unquote(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}
