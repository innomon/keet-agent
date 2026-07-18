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
	if cfg.StorageDir != "storage" {
		t.Errorf("expected default StorageDir 'storage', got %q", cfg.StorageDir)
	}
	if cfg.DBHost != "localhost" {
		t.Errorf("expected default DBHost 'localhost', got %q", cfg.DBHost)
	}
	if cfg.DBPort != "5432" {
		t.Errorf("expected default DBPort '5432', got %q", cfg.DBPort)
	}
	if cfg.DBUser != "postgres" {
		t.Errorf("expected default DBUser 'postgres', got %q", cfg.DBUser)
	}
	if cfg.DBPassword != "postgres" {
		t.Errorf("expected default DBPassword 'postgres', got %q", cfg.DBPassword)
	}
	if cfg.DBName != "keet_gateway" {
		t.Errorf("expected default DBName 'keet_gateway', got %q", cfg.DBName)
	}
	if cfg.DBSSLMode != "disable" {
		t.Errorf("expected default DBSSLMode 'disable', got %q", cfg.DBSSLMode)
	}
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	os.Setenv("LOG_LEVEL", "DEBUG")
	os.Setenv("CONSOLE_LOG_ENABLED", "false")
	os.Setenv("LOG_MAX_SIZE_MB", "25")
	os.Setenv("SOCKET_PATH", "/tmp/custom.sock")
	os.Setenv("STORAGE_DIR", "custom_storage")
	os.Setenv("DB_HOST", "db.example.com")
	os.Setenv("DB_PORT", "5433")
	os.Setenv("DB_USER", "custom_user")
	os.Setenv("DB_PASSWORD", "secret")
	os.Setenv("DB_NAME", "custom_db")
	os.Setenv("DB_SSLMODE", "require")

	defer func() {
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("CONSOLE_LOG_ENABLED")
		os.Unsetenv("LOG_MAX_SIZE_MB")
		os.Unsetenv("SOCKET_PATH")
		os.Unsetenv("STORAGE_DIR")
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("DB_SSLMODE")
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
	if cfg.StorageDir != "custom_storage" {
		t.Errorf("expected StorageDir 'custom_storage', got %q", cfg.StorageDir)
	}
	if cfg.DBHost != "db.example.com" {
		t.Errorf("expected DBHost 'db.example.com', got %q", cfg.DBHost)
	}
	if cfg.DBPort != "5433" {
		t.Errorf("expected DBPort '5433', got %q", cfg.DBPort)
	}
	if cfg.DBUser != "custom_user" {
		t.Errorf("expected DBUser 'custom_user', got %q", cfg.DBUser)
	}
	if cfg.DBPassword != "secret" {
		t.Errorf("expected DBPassword 'secret', got %q", cfg.DBPassword)
	}
	if cfg.DBName != "custom_db" {
		t.Errorf("expected DBName 'custom_db', got %q", cfg.DBName)
	}
	if cfg.DBSSLMode != "require" {
		t.Errorf("expected DBSSLMode 'require', got %q", cfg.DBSSLMode)
	}
}
