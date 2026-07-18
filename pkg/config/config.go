package config

import (
	"os"
	"strconv"
)

type Config struct {
	LogLevel       string
	ConsoleEnabled bool
	FileEnabled    bool
	LogDir         string
	LogFileName    string
	LogMaxSizeMB   int
	LogMaxBackups  int
	SocketPath     string
}

func LoadConfig() Config {
	return Config{
		LogLevel:       getEnv("LOG_LEVEL", "INFO"),
		ConsoleEnabled: getEnvBool("CONSOLE_LOG_ENABLED", true),
		FileEnabled:    getEnvBool("FILE_LOG_ENABLED", true),
		LogDir:         getEnv("LOG_DIR", "logs"),
		LogFileName:    getEnv("LOG_FILE_NAME", "gateway.log"),
		LogMaxSizeMB:   getEnvInt("LOG_MAX_SIZE_MB", 10),
		LogMaxBackups:  getEnvInt("LOG_MAX_BACKUPS", 5),
		SocketPath:     getEnv("SOCKET_PATH", "/tmp/keet-adk.sock"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		b, err := strconv.ParseBool(value)
		if err == nil {
			return b
		}
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		i, err := strconv.Atoi(value)
		if err == nil {
			return i
		}
	}
	return fallback
}
