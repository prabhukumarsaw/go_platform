package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration, sourced from environment variables.
type Config struct {
	Server     ServerConfig
	Postgres   PostgresConfig
	Redis      RedisConfig
	Meili      MeiliConfig
	JWT        JWTConfig
	Media      MediaConfig
	GoogleOAuth GoogleOAuthConfig
	OTP        OTPConfig
}

type ServerConfig struct {
	Port string
	Env  string // development, staging, production
}

type PostgresConfig struct {
	Host     string
	Port     string
	DB       string
	User     string
	Password string
	SSLMode  string
}

// DSN returns the PostgreSQL connection string.
func (p PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		p.User, p.Password, p.Host, p.Port, p.DB, p.SSLMode,
	)
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type MeiliConfig struct {
	Host      string
	MasterKey string
}

type JWTConfig struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

type MediaConfig struct {
	UploadDir   string // local filesystem path for media storage
	MaxFileSize int64  // bytes
	BaseURL     string // public URL prefix for serving media
}

type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type OTPConfig struct {
	Expiry      time.Duration
	MaxAttempts int
	SMSProvider string // "console" for dev (logs to stdout), "twilio", etc.
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	loadDotEnv(".env")

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Env:  getEnv("SERVER_ENV", "development"),
		},
		Postgres: PostgresConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			DB:       getEnv("POSTGRES_DB", "news_db"),
			User:     getEnv("POSTGRES_USER", "postgres"),
			Password: getEnv("POSTGRES_PASSWORD", "12345"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Meili: MeiliConfig{
			Host:      getEnv("MEILI_HOST", "http://localhost:7700"),
			MasterKey: getEnv("MEILI_MASTER_KEY", "masterkey_dev_only"),
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", "change_me_in_production"),
			AccessExpiry:  getEnvDuration("JWT_ACCESS_EXPIRY", 15*time.Minute),
			RefreshExpiry: getEnvDuration("JWT_REFRESH_EXPIRY", 168*time.Hour),
		},
		Media: MediaConfig{
			UploadDir:   getEnv("MEDIA_UPLOAD_DIR", "./uploads"),
			MaxFileSize: getEnvInt64("MEDIA_MAX_FILE_SIZE", 20*1024*1024), // 20MB
			BaseURL:     getEnv("MEDIA_BASE_URL", "http://localhost:8080/uploads"),
		},
		GoogleOAuth: GoogleOAuthConfig{
			ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
			ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/v1/auth/google/callback"),
		},
		OTP: OTPConfig{
			Expiry:      getEnvDuration("OTP_EXPIRY", 5*time.Minute),
			MaxAttempts: getEnvInt("OTP_MAX_ATTEMPTS", 3),
			SMSProvider: getEnv("SMS_PROVIDER", "console"),
		},
	}

	return cfg, nil
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

func getEnvInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func loadDotEnv(filepath string) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(parts[1])
			v = strings.Trim(v, `"'`)
			if os.Getenv(k) == "" {
				_ = os.Setenv(k, v)
			}
		}
	}
}
