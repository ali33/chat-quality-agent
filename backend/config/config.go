package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	// Server
	ServerPort string
	ServerHost string

	// Database — DBDriver is normalized: sqlite | mysql | postgres | sqlserver
	DBDriver   string
	SQLitePath string // when DBDriver == sqlite
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string // postgres only, e.g. disable, require, verify-full

	MessageDataDir string // daily JSONL files: YYYY-MM-DD.jsonl

	// Security
	JWTSecret     string
	EncryptionKey string // 32 bytes for AES-256-GCM

	// Rate limiting
	RateLimitPerIP   int // requests per minute
	RateLimitPerUser int

	// AI
	AIMaxTokens int // max tokens for AI responses

	// Environment
	Env string // "development" | "production"
}

// NormalizeDBDriver maps env aliases to a canonical driver name.
func NormalizeDBDriver(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch s {
	case "", "sqlite", "sqlite3":
		return "sqlite"
	case "mysql", "mariadb":
		return "mysql"
	case "postgres", "postgresql", "pg":
		return "postgres"
	case "sqlserver", "mssql", "microsoft":
		return "sqlserver"
	default:
		return ""
	}
}

func Load() (*Config, error) {
	rawDriver := getEnv("DB_DRIVER", "sqlite")
	norm := NormalizeDBDriver(rawDriver)
	if norm == "" {
		return nil, fmt.Errorf("unsupported DB_DRIVER %q: use sqlite, mysql, postgres, or sqlserver", strings.TrimSpace(rawDriver))
	}

	cfg := &Config{
		ServerPort:       getEnv("SERVER_PORT", "8080"),
		ServerHost:       getEnv("SERVER_HOST", "127.0.0.1"),
		DBDriver:         norm,
		SQLitePath:       getEnv("SQLITE_PATH", "data/cqa.db"),
		DBHost:           getEnv("DB_HOST", ""),
		DBPort:           getEnv("DB_PORT", ""),
		DBUser:           getEnv("DB_USER", ""),
		DBPassword:       getEnv("DB_PASSWORD", ""),
		DBName:           getEnv("DB_NAME", "cqa"),
		DBSSLMode:        getEnv("DB_SSLMODE", "disable"),
		MessageDataDir:   getEnv("MESSAGE_DATA_DIR", "data/messages"),
		JWTSecret:        getEnv("JWT_SECRET", ""),
		EncryptionKey:    getEnv("ENCRYPTION_KEY", ""),
		RateLimitPerIP:   getEnvInt("RATE_LIMIT_PER_IP", 500),
		RateLimitPerUser: getEnvInt("RATE_LIMIT_PER_USER", 1000),
		AIMaxTokens:      getEnvInt("AI_MAX_TOKENS", 16384),
		Env:              getEnv("APP_ENV", "development"),
	}

	if cfg.DBDriver != "sqlite" {
		if cfg.DBPort == "" {
			switch cfg.DBDriver {
			case "mysql":
				cfg.DBPort = "3306"
			case "postgres":
				cfg.DBPort = "5432"
			case "sqlserver":
				cfg.DBPort = "1433"
			}
		}
		if cfg.DBUser == "" {
			return nil, fmt.Errorf("DB_USER is required when DB_DRIVER=%s", cfg.DBDriver)
		}
		if cfg.DBName == "" {
			return nil, fmt.Errorf("DB_NAME is required when DB_DRIVER=%s", cfg.DBDriver)
		}
		if cfg.DBHost == "" {
			return nil, fmt.Errorf("DB_HOST is required when DB_DRIVER=%s", cfg.DBDriver)
		}
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters for HS256 security")
	}
	if cfg.EncryptionKey == "" {
		return nil, fmt.Errorf("ENCRYPTION_KEY is required")
	}
	if len(cfg.EncryptionKey) != 32 {
		return nil, fmt.Errorf("ENCRYPTION_KEY must be exactly 32 bytes for AES-256-GCM")
	}

	return cfg, nil
}

// MessageTimeLocation is used to pick the calendar day for daily message files (env TZ when set).
func (*Config) MessageTimeLocation() *time.Location {
	if tz := os.Getenv("TZ"); tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			return loc
		}
	}
	return time.Local
}

func (c *Config) ListenAddr() string {
	return fmt.Sprintf("%s:%s", c.ServerHost, c.ServerPort)
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

// StaticDir is the folder with production UI (index.html, assets/, …).
// Set STATIC_DIR to override; otherwise it is <directory of executable>/static
// so the server works no matter the current working directory.
func StaticDir() string {
	if s := strings.TrimSpace(os.Getenv("STATIC_DIR")); s != "" {
		return filepath.Clean(s)
	}
	exe, err := os.Executable()
	if err != nil {
		return "static"
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Join(filepath.Dir(exe), "static")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
