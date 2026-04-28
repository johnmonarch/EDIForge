package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	ProjectConfigFile = "edi-json.yml"
	UserConfigFile    = "config.yml"
	UserConfigDir     = ".edi-json"
)

type configPatch struct {
	Server      serverPatch
	Translation translationPatch
	Schemas     schemasPatch
	Privacy     privacyPatch
	Limits      limitsPatch
}

type serverPatch struct {
	Host                         *string
	Port                         *int
	RequireToken                 *bool
	RequireTokenOutsideLocalhost *bool
	MaxBodyMB                    *int64
	CORSOrigin                   *string
}

type translationPatch struct {
	DefaultMode        *string
	IncludeEnvelope    *bool
	IncludeRawSegments *bool
}

type schemasPatch struct {
	Paths *[]string
}

type privacyPatch struct {
	StoreHistory *bool
	Telemetry    *bool
}

type limitsPatch struct {
	MaxFileSizeMB *int64
}

// Load reads the default user and project configuration files when present.
func Load() (Config, error) {
	userPath := ""
	if home, err := os.UserHomeDir(); err == nil {
		userPath = filepath.Join(home, UserConfigDir, UserConfigFile)
	}
	return LoadFromPaths(userPath, ProjectConfigFile)
}

// LoadFromPaths starts with Default and merges optional user then project files.
func LoadFromPaths(userPath, projectPath string) (Config, error) {
	cfg := Default()
	userPatch, err := readConfigPatch(userPath)
	if err != nil {
		return Config{}, err
	}
	projectPatch, err := readConfigPatch(projectPath)
	if err != nil {
		return Config{}, err
	}
	applyPatch(&cfg, userPatch)
	applyPatch(&cfg, projectPatch)
	mergeSchemaPaths(&cfg, userPatch, projectPatch)
	return cfg, nil
}

func readConfigPatch(path string) (configPatch, error) {
	if path == "" {
		return configPatch{}, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return configPatch{}, nil
	}
	if err != nil {
		return configPatch{}, err
	}
	patch, err := parseConfigYAML(string(data), path)
	if err != nil {
		return configPatch{}, err
	}
	return patch, nil
}

func mergeSchemaPaths(cfg *Config, userPatch, projectPatch configPatch) {
	if userPatch.Schemas.Paths == nil && projectPatch.Schemas.Paths == nil {
		return
	}
	paths := []string{}
	if projectPatch.Schemas.Paths != nil {
		paths = append(paths, (*projectPatch.Schemas.Paths)...)
	}
	if userPatch.Schemas.Paths != nil {
		paths = append(paths, (*userPatch.Schemas.Paths)...)
	}
	cfg.Schemas.Paths = paths
}

func applyPatch(cfg *Config, patch configPatch) {
	if patch.Server.Host != nil {
		cfg.Server.Host = *patch.Server.Host
	}
	if patch.Server.Port != nil {
		cfg.Server.Port = *patch.Server.Port
	}
	if patch.Server.RequireToken != nil {
		cfg.Server.RequireToken = *patch.Server.RequireToken
	}
	if patch.Server.RequireTokenOutsideLocalhost != nil {
		cfg.Server.RequireTokenOutsideLocalhost = *patch.Server.RequireTokenOutsideLocalhost
	}
	if patch.Server.MaxBodyMB != nil {
		cfg.Server.MaxBodyMB = *patch.Server.MaxBodyMB
	}
	if patch.Server.CORSOrigin != nil {
		cfg.Server.CORSOrigin = *patch.Server.CORSOrigin
	}
	if patch.Translation.DefaultMode != nil {
		cfg.Translation.DefaultMode = *patch.Translation.DefaultMode
	}
	if patch.Translation.IncludeEnvelope != nil {
		cfg.Translation.IncludeEnvelope = *patch.Translation.IncludeEnvelope
	}
	if patch.Translation.IncludeRawSegments != nil {
		cfg.Translation.IncludeRawSegments = *patch.Translation.IncludeRawSegments
	}
	if patch.Schemas.Paths != nil {
		cfg.Schemas.Paths = append([]string(nil), (*patch.Schemas.Paths)...)
	}
	if patch.Privacy.StoreHistory != nil {
		cfg.Privacy.StoreHistory = *patch.Privacy.StoreHistory
	}
	if patch.Privacy.Telemetry != nil {
		cfg.Privacy.Telemetry = *patch.Privacy.Telemetry
	}
	if patch.Limits.MaxFileSizeMB != nil {
		cfg.Limits.MaxFileSizeMB = *patch.Limits.MaxFileSizeMB
	}
}

func parseConfigYAML(input, path string) (configPatch, error) {
	var patch configPatch
	var section string
	var listKey string
	scanner := bufio.NewScanner(strings.NewReader(input))
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := stripYAMLComment(scanner.Text())
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent == 0 {
			listKey = ""
			key, value, ok := splitYAMLKV(trimmed)
			if !ok {
				continue
			}
			if value == "" {
				section = key
				continue
			}
			section = ""
			continue
		}
		if section == "" {
			continue
		}
		if listKey == "schemas.paths" && strings.HasPrefix(trimmed, "- ") {
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			if value == "" {
				return configPatch{}, configLineError(path, lineNo, "schemas.paths item cannot be empty", nil)
			}
			appendSchemaPath(&patch, unquote(value))
			continue
		}
		listKey = ""
		key, value, ok := splitYAMLKV(trimmed)
		if !ok {
			continue
		}
		if err := applyYAMLValue(&patch, section, key, value, path, lineNo, &listKey); err != nil {
			return configPatch{}, err
		}
	}
	if err := scanner.Err(); err != nil {
		return configPatch{}, err
	}
	return patch, nil
}

func applyYAMLValue(patch *configPatch, section, key, value, path string, lineNo int, listKey *string) error {
	switch section {
	case "server":
		return applyServerValue(patch, key, value, path, lineNo)
	case "translation":
		return applyTranslationValue(patch, key, value, path, lineNo)
	case "schemas":
		if key != "paths" {
			return nil
		}
		if value == "" {
			*listKey = "schemas.paths"
			paths := []string{}
			patch.Schemas.Paths = &paths
			return nil
		}
		paths, err := parseStringList(value)
		if err != nil {
			return configLineError(path, lineNo, "invalid schemas.paths", err)
		}
		patch.Schemas.Paths = &paths
		return nil
	case "privacy":
		return applyPrivacyValue(patch, key, value, path, lineNo)
	case "limits":
		return applyLimitsValue(patch, key, value, path, lineNo)
	default:
		return nil
	}
}

func applyServerValue(patch *configPatch, key, value, path string, lineNo int) error {
	switch key {
	case "host":
		host := unquote(value)
		patch.Server.Host = &host
	case "port":
		port64, err := strconv.ParseInt(unquote(value), 10, 0)
		if err != nil {
			return configLineError(path, lineNo, "invalid server.port", err)
		}
		port := int(port64)
		patch.Server.Port = &port
	case "requireToken":
		requireToken, err := parseBool(value)
		if err != nil {
			return configLineError(path, lineNo, "invalid server.requireToken", err)
		}
		patch.Server.RequireToken = &requireToken
	case "requireTokenOutsideLocalhost":
		requireToken, err := parseBool(value)
		if err != nil {
			return configLineError(path, lineNo, "invalid server.requireTokenOutsideLocalhost", err)
		}
		patch.Server.RequireTokenOutsideLocalhost = &requireToken
	case "maxBodyMb":
		maxBodyMB, err := parseInt64(value)
		if err != nil {
			return configLineError(path, lineNo, "invalid server.maxBodyMb", err)
		}
		patch.Server.MaxBodyMB = &maxBodyMB
	case "corsOrigin":
		corsOrigin := unquote(value)
		patch.Server.CORSOrigin = &corsOrigin
	}
	return nil
}

func applyTranslationValue(patch *configPatch, key, value, path string, lineNo int) error {
	switch key {
	case "defaultMode":
		mode := unquote(value)
		patch.Translation.DefaultMode = &mode
	case "includeEnvelope":
		includeEnvelope, err := parseBool(value)
		if err != nil {
			return configLineError(path, lineNo, "invalid translation.includeEnvelope", err)
		}
		patch.Translation.IncludeEnvelope = &includeEnvelope
	case "includeRawSegments":
		includeRawSegments, err := parseBool(value)
		if err != nil {
			return configLineError(path, lineNo, "invalid translation.includeRawSegments", err)
		}
		patch.Translation.IncludeRawSegments = &includeRawSegments
	}
	return nil
}

func applyPrivacyValue(patch *configPatch, key, value, path string, lineNo int) error {
	switch key {
	case "storeHistory":
		storeHistory, err := parseBool(value)
		if err != nil {
			return configLineError(path, lineNo, "invalid privacy.storeHistory", err)
		}
		patch.Privacy.StoreHistory = &storeHistory
	case "telemetry":
		telemetry, err := parseBool(value)
		if err != nil {
			return configLineError(path, lineNo, "invalid privacy.telemetry", err)
		}
		patch.Privacy.Telemetry = &telemetry
	}
	return nil
}

func applyLimitsValue(patch *configPatch, key, value, path string, lineNo int) error {
	if key != "maxFileSizeMb" {
		return nil
	}
	maxFileSizeMB, err := parseInt64(value)
	if err != nil {
		return configLineError(path, lineNo, "invalid limits.maxFileSizeMb", err)
	}
	patch.Limits.MaxFileSizeMB = &maxFileSizeMB
	return nil
}

func appendSchemaPath(patch *configPatch, path string) {
	if patch.Schemas.Paths == nil {
		paths := []string{}
		patch.Schemas.Paths = &paths
	}
	*patch.Schemas.Paths = append(*patch.Schemas.Paths, path)
}

func parseBool(value string) (bool, error) {
	return strconv.ParseBool(unquote(value))
}

func parseInt64(value string) (int64, error) {
	return strconv.ParseInt(unquote(value), 10, 64)
}

func parseStringList(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		value = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
		if value == "" {
			return []string{}, nil
		}
		parts := strings.Split(value, ",")
		paths := make([]string, 0, len(parts))
		for _, part := range parts {
			item := unquote(strings.TrimSpace(part))
			if item == "" {
				return nil, fmt.Errorf("empty list item")
			}
			paths = append(paths, item)
		}
		return paths, nil
	}
	return []string{unquote(value)}, nil
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

func stripYAMLComment(line string) string {
	inSingleQuote := false
	inDoubleQuote := false
	for i, r := range line {
		switch r {
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
		case '#':
			if !inSingleQuote && !inDoubleQuote {
				return line[:i]
			}
		}
	}
	return line
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

func configLineError(path string, lineNo int, message string, err error) error {
	if err == nil {
		return fmt.Errorf("%s:%d: %s", path, lineNo, message)
	}
	return fmt.Errorf("%s:%d: %s: %w", path, lineNo, message, err)
}
