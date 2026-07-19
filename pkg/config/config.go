package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	LogLevel       string   `yaml:"log_level"`
	ConsoleEnabled bool     `yaml:"console_log_enabled"`
	FileEnabled    bool     `yaml:"file_log_enabled"`
	LogDir         string   `yaml:"log_dir"`
	LogFileName    string   `yaml:"log_file_name"`
	LogMaxSizeMB   int      `yaml:"log_max_size_mb"`
	LogMaxBackups  int      `yaml:"log_max_backups"`
	SocketPath     string   `yaml:"socket_path"`
	StorageDir     string   `yaml:"storage_dir"`
	DBType         string   `yaml:"db_type"` // "postgres" or "bbolt"
	BBoltPath      string   `yaml:"bbolt_path"` // path to bbolt file (e.g. "storage/gateway.db")
	DBHost         string   `yaml:"db_host"`
	DBPort         string   `yaml:"db_port"`
	DBUser         string   `yaml:"db_user"`
	DBPassword     string   `yaml:"db_password"`
	DBName         string   `yaml:"db_name"`
	DBSSLMode      string   `yaml:"db_sslmode"`
	P2PPort        string   `yaml:"p2p_port"`
	P2PListenAddr  string   `yaml:"p2p_listen_addr"`
	ClientWhitelist []string `yaml:"client_whitelist"`
}

func LoadConfig() Config {
	// 1. Initialize with environment variables and defaults
	cfg := Config{
		LogLevel:        getEnv("LOG_LEVEL", "INFO"),
		ConsoleEnabled:  getEnvBool("CONSOLE_LOG_ENABLED", true),
		FileEnabled:     getEnvBool("FILE_LOG_ENABLED", true),
		LogDir:          getEnv("LOG_DIR", "logs"),
		LogFileName:     getEnv("LOG_FILE_NAME", "gateway.log"),
		LogMaxSizeMB:    getEnvInt("LOG_MAX_SIZE_MB", 10),
		LogMaxBackups:   getEnvInt("LOG_MAX_BACKUPS", 5),
		SocketPath:      getEnv("SOCKET_PATH", "/tmp/keet-adk.sock"),
		StorageDir:      getEnv("STORAGE_DIR", "storage"),
		DBType:          getEnv("DB_TYPE", "bbolt"),
		BBoltPath:       getEnv("BBOLT_PATH", "storage/gateway.db"),
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBUser:          getEnv("DB_USER", "postgres"),
		DBPassword:      getEnv("DB_PASSWORD", "postgres"),
		DBName:          getEnv("DB_NAME", "keet_gateway"),
		DBSSLMode:       getEnv("DB_SSLMODE", "disable"),
		P2PPort:         getEnv("P2P_PORT", "0"),
		P2PListenAddr:   getEnv("P2P_LISTEN_ADDR", "127.0.0.1"),
		ClientWhitelist: getEnvSlice("CLIENT_WHITELIST", nil),
	}

	// 2. Find config.yaml path in order of precedence:
	//    a. Command-line argument: --config path/to/config.yaml or -config path/to/config.yaml
	//    b. Current directory: ./config.yaml
	//    c. Executable's directory: path/to/executable/config.yaml
	var path string

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "--config" || arg == "-config" {
			if i+1 < len(os.Args) {
				path = os.Args[i+1]
				break
			}
		} else if strings.HasPrefix(arg, "--config=") {
			path = strings.TrimPrefix(arg, "--config=")
			break
		} else if strings.HasPrefix(arg, "-config=") {
			path = strings.TrimPrefix(arg, "-config=")
			break
		}
	}

	if path == "" {
		if _, err := os.Stat("config.yaml"); err == nil {
			path = "config.yaml"
		}
	}

	if path == "" {
		if execPath, err := os.Executable(); err == nil {
			execDir := filepath.Dir(execPath)
			execConfig := filepath.Join(execDir, "config.yaml")
			if _, err := os.Stat(execConfig); err == nil {
				path = execConfig
			}
		}
	}

	// 3. Parse and overlay yaml configuration if found
	if path != "" {
		file, err := os.Open(path)
		if err == nil {
			defer file.Close()
			decoder := yaml.NewDecoder(file)
			if err := decoder.Decode(&cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to decode config file %s: %v\n", path, err)
			} else {
				fmt.Printf("Successfully loaded configuration from %s\n", path)
			}
		}
	}

	return cfg
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

func getEnvSlice(key string, fallback []string) []string {
	if value, exists := os.LookupEnv(key); exists {
		if value == "" {
			return []string{}
		}
		parts := strings.Split(value, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts
	}
	return fallback
}


