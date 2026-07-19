package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_EnvFallback(t *testing.T) {
	// Clean up environment variables to test defaults
	os.Unsetenv("DB_TYPE")
	os.Unsetenv("LOG_LEVEL")

	// Verify default env-based loading works
	cfg := LoadConfig()
	if cfg.DBType != "bbolt" {
		t.Errorf("expected default DBType to be bbolt, got %q", cfg.DBType)
	}
	if cfg.LogLevel != "INFO" {
		t.Errorf("expected default LogLevel to be INFO, got %q", cfg.LogLevel)
	}
}

func TestLoadConfig_YamlParsing(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	yamlPath := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `
log_level: DEBUG
db_type: postgres
storage_dir: custom_storage
`
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	// 1. Test CLI argument lookup
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd", "--config", yamlPath}

	cfg := LoadConfig()
	if cfg.LogLevel != "DEBUG" {
		t.Errorf("expected LogLevel to be DEBUG from YAML, got %q", cfg.LogLevel)
	}
	if cfg.DBType != "postgres" {
		t.Errorf("expected DBType to be postgres from YAML, got %q", cfg.DBType)
	}
	if cfg.StorageDir != "custom_storage" {
		t.Errorf("expected StorageDir to be custom_storage from YAML, got %q", cfg.StorageDir)
	}

	// Test flag formats: --config=
	os.Args = []string{"cmd", "--config=" + yamlPath}
	cfg = LoadConfig()
	if cfg.LogLevel != "DEBUG" {
		t.Errorf("expected LogLevel DEBUG from --config= format, got %q", cfg.LogLevel)
	}
}

func TestLoadConfig_CurrentDirLookup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config-test-curr-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working dir: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change working dir: %v", err)
	}

	yamlContent := `
log_level: WARN
socket_path: /tmp/custom-socket.sock
`
	if err := os.WriteFile("config.yaml", []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write config.yaml: %v", err)
	}

	// Unset CLI args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd"}

	cfg := LoadConfig()
	if cfg.LogLevel != "WARN" {
		t.Errorf("expected LogLevel WARN from current dir lookup, got %q", cfg.LogLevel)
	}
	if cfg.SocketPath != "/tmp/custom-socket.sock" {
		t.Errorf("expected SocketPath /tmp/custom-socket.sock, got %q", cfg.SocketPath)
	}
}

func TestHelpers(t *testing.T) {
	os.Setenv("TEST_BOOL_TRUE", "true")
	os.Setenv("TEST_BOOL_FALSE", "false")
	os.Setenv("TEST_BOOL_INVALID", "not-a-bool")
	os.Setenv("TEST_INT", "123")
	os.Setenv("TEST_INT_INVALID", "not-an-int")
	os.Setenv("TEST_SLICE", "alice, bob, charlie")
	os.Setenv("TEST_SLICE_EMPTY", "")

	defer func() {
		os.Unsetenv("TEST_BOOL_TRUE")
		os.Unsetenv("TEST_BOOL_FALSE")
		os.Unsetenv("TEST_BOOL_INVALID")
		os.Unsetenv("TEST_INT")
		os.Unsetenv("TEST_INT_INVALID")
		os.Unsetenv("TEST_SLICE")
		os.Unsetenv("TEST_SLICE_EMPTY")
	}()

	if !getEnvBool("TEST_BOOL_TRUE", false) {
		t.Error("expected true")
	}
	if getEnvBool("TEST_BOOL_FALSE", true) {
		t.Error("expected false")
	}
	if !getEnvBool("TEST_BOOL_INVALID", true) {
		t.Error("expected default true")
	}
	if getEnvInt("TEST_INT", 0) != 123 {
		t.Error("expected 123")
	}
	if getEnvInt("TEST_INT_INVALID", 999) != 999 {
		t.Error("expected default 999")
	}

	slice := getEnvSlice("TEST_SLICE", nil)
	if len(slice) != 3 || slice[0] != "alice" || slice[1] != "bob" || slice[2] != "charlie" {
		t.Errorf("expected slice [alice, bob, charlie], got %v", slice)
	}

	emptySlice := getEnvSlice("TEST_SLICE_EMPTY", []string{"default"})
	if len(emptySlice) != 0 {
		t.Errorf("expected empty slice, got %v", emptySlice)
	}

	fallbackSlice := getEnvSlice("TEST_SLICE_MISSING", []string{"fallback"})
	if len(fallbackSlice) != 1 || fallbackSlice[0] != "fallback" {
		t.Errorf("expected fallback slice, got %v", fallbackSlice)
	}
}


