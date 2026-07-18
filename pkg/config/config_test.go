package config

import (
	"os"
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	cfg := LoadConfig()

	if cfg.LogLevel != "INFO" {
		t.Errorf("expected default LogLevel 'INFO', got %q", cfg.LogLevel)
	}
	if !cfg.ConsoleEnabled {
		t.Error("expected default ConsoleEnabled true")
	}
	if !cfg.FileEnabled {
		t.Error("expected default FileEnabled true")
	}
	if cfg.LogDir != "logs" {
		t.Errorf("expected default LogDir 'logs', got %q", cfg.LogDir)
	}
	if cfg.LogFileName != "gateway.log" {
		t.Errorf("expected default LogFileName 'gateway.log', got %q", cfg.LogFileName)
	}
	if cfg.LogMaxSizeMB != 10 {
		t.Errorf("expected default LogMaxSizeMB 10, got %d", cfg.LogMaxSizeMB)
	}
	if cfg.LogMaxBackups != 5 {
		t.Errorf("expected default LogMaxBackups 5, got %d", cfg.LogMaxBackups)
	}
	if cfg.SocketPath != "/tmp/keet-adk.sock" {
		t.Errorf("expected default SocketPath '/tmp/keet-adk.sock', got %q", cfg.SocketPath)
	}
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	os.Setenv("LOG_LEVEL", "DEBUG")
	os.Setenv("CONSOLE_LOG_ENABLED", "false")
	os.Setenv("LOG_MAX_SIZE_MB", "25")
	os.Setenv("SOCKET_PATH", "/tmp/custom.sock")

	defer func() {
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("CONSOLE_LOG_ENABLED")
		os.Unsetenv("LOG_MAX_SIZE_MB")
		os.Unsetenv("SOCKET_PATH")
	}()

	cfg := LoadConfig()

	if cfg.LogLevel != "DEBUG" {
		t.Errorf("expected LogLevel 'DEBUG', got %q", cfg.LogLevel)
	}
	if cfg.ConsoleEnabled {
		t.Error("expected ConsoleEnabled false")
	}
	if cfg.LogMaxSizeMB != 25 {
		t.Errorf("expected LogMaxSizeMB 25, got %d", cfg.LogMaxSizeMB)
	}
	if cfg.SocketPath != "/tmp/custom.sock" {
		t.Errorf("expected SocketPath '/tmp/custom.sock', got %q", cfg.SocketPath)
	}
}
