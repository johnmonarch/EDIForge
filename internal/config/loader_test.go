package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadFromPathsMissingFilesReturnsDefaults(t *testing.T) {
	cfg, err := LoadFromPaths("", filepath.Join(t.TempDir(), "missing.yml"))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Fatalf("server.host = %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 8765 {
		t.Fatalf("server.port = %d", cfg.Server.Port)
	}
	if !cfg.Server.RequireTokenOutsideLocalhost {
		t.Fatalf("server.requireTokenOutsideLocalhost = false, want true")
	}
	if cfg.Server.MaxBodyMB != 50 || cfg.Limits.MaxFileSizeMB != 50 {
		t.Fatalf("limits = server %d file %d", cfg.Server.MaxBodyMB, cfg.Limits.MaxFileSizeMB)
	}
	if cfg.Translation.DefaultMode != "structural" || !cfg.Translation.IncludeEnvelope || cfg.Translation.IncludeRawSegments {
		t.Fatalf("translation defaults = %+v", cfg.Translation)
	}
	if cfg.Privacy.StoreHistory || cfg.Privacy.Telemetry {
		t.Fatalf("privacy defaults = %+v", cfg.Privacy)
	}
}

func TestLoadFromPathsMergesUserThenProjectConfig(t *testing.T) {
	dir := t.TempDir()
	userPath := filepath.Join(dir, "home", UserConfigDir, UserConfigFile)
	projectPath := filepath.Join(dir, ProjectConfigFile)
	writeConfig(t, userPath, `
server:
  host: 0.0.0.0 # user bind
  port: 9000
  requireToken: true
  requireTokenOutsideLocalhost: false
  maxBodyMb: 25
translation:
  defaultMode: annotated
  includeEnvelope: false
  includeRawSegments: true
schemas:
  paths:
    - /user/schemas # user schemas
privacy:
  storeHistory: true
  telemetry: true
limits:
  maxFileSizeMb: 10
`)
	writeConfig(t, projectPath, `
server:
  port: 7654
  requireToken: false
translation:
  includeRawSegments: false
schemas:
  paths:
    - /project/schemas
privacy:
  telemetry: false
limits:
  maxFileSizeMb: 75
`)

	cfg, err := LoadFromPaths(userPath, projectPath)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Server.Host != "0.0.0.0" {
		t.Fatalf("server.host = %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 7654 {
		t.Fatalf("server.port = %d", cfg.Server.Port)
	}
	if cfg.Server.RequireToken {
		t.Fatalf("server.requireToken = true, want project override false")
	}
	if cfg.Server.RequireTokenOutsideLocalhost {
		t.Fatalf("server.requireTokenOutsideLocalhost = true, want user override false")
	}
	if cfg.Server.MaxBodyMB != 25 {
		t.Fatalf("server.maxBodyMb = %d", cfg.Server.MaxBodyMB)
	}
	if cfg.Translation.DefaultMode != "annotated" || cfg.Translation.IncludeEnvelope || cfg.Translation.IncludeRawSegments {
		t.Fatalf("translation = %+v", cfg.Translation)
	}
	wantPaths := []string{"/project/schemas", "/user/schemas"}
	if !reflect.DeepEqual(cfg.Schemas.Paths, wantPaths) {
		t.Fatalf("schemas.paths = %#v, want %#v", cfg.Schemas.Paths, wantPaths)
	}
	if !cfg.Privacy.StoreHistory || cfg.Privacy.Telemetry {
		t.Fatalf("privacy = %+v", cfg.Privacy)
	}
	if cfg.Limits.MaxFileSizeMB != 75 {
		t.Fatalf("limits.maxFileSizeMb = %d", cfg.Limits.MaxFileSizeMB)
	}
}

func TestLoadFromPathsReportsInvalidTypedValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), ProjectConfigFile)
	writeConfig(t, path, `
server:
  port: nope
`)

	_, err := LoadFromPaths("", path)
	if err == nil {
		t.Fatal("LoadFromPaths error = nil")
	}
	if !strings.Contains(err.Error(), "invalid server.port") {
		t.Fatalf("error = %v", err)
	}
}

func TestNewSchemaRegistryPrependsConfiguredRootsAndKeepsBuiltInFallback(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("HOME", home)

	registry := NewSchemaRegistry(Config{
		Schemas: SchemaConfig{Paths: []string{"~/schemas", "schemas/examples", "", "~/schemas"}},
	})

	want := []string{filepath.Join(home, "schemas"), "schemas/examples"}
	if !reflect.DeepEqual(registry.Roots, want) {
		t.Fatalf("registry roots = %#v, want %#v", registry.Roots, want)
	}
}

func writeConfig(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
