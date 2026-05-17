// Package config loads typed configuration from a .env file (when present) and
// the process environment. Environment variables always win over .env values;
// .env values win over the in-code defaults. This mirrors spring-dotenv's lookup
// order.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App        AppConfig
	Server     ServerConfig
	DB         DBConfig
	Log        LogConfig
	JWT        JWTConfig
	Sessions   SessionsConfig
	CORS       CORSConfig
	RateLimit  RateLimitConfig
	Idem       IdempotencyConfig
	Audit      AuditConfig
	Storage    StorageConfig
	Mail       MailConfig
	Bootstrap  BootstrapConfig
	Migrations MigrationsConfig
}

type AppConfig struct {
	Name string
	Env  string // dev | prod
}

type ServerConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type DBConfig struct {
	Driver          string // postgres | mysql
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string // postgres only
	TimeZone        string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type LogConfig struct {
	Level      string
	Format     string // console | json
	Dir        string
	File       string // log filename inside Dir (e.g. "app.log"); empty disables file output
	MaxSizeMB  int    // rotate when the active log reaches this size (megabytes)
	MaxBackups int    // max rotated files to keep before deletion (0 = unlimited)
	MaxAgeDays int    // delete rotated files older than this many days (0 = no age limit)
	Compress   bool   // gzip rotated files
}

type JWTConfig struct {
	Secret     string
	Issuer     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type SessionsConfig struct {
	CleanupInterval  time.Duration
	CleanupRetention time.Duration
}

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           time.Duration
}

type RateLimitConfig struct {
	Enabled          bool
	Capacity         int
	RefillTokens     int
	RefillPeriod     time.Duration
	AuthCapacity     int
	AuthRefillTokens int
	AuthRefillPeriod time.Duration
	AuthPathPrefixes []string
	IncludePrefixes  []string
}

type IdempotencyConfig struct {
	Enabled         bool
	TTL             time.Duration
	CleanupInterval time.Duration
}

type AuditConfig struct {
	Enabled             bool
	ExposeAPI           bool
	AuditAll            bool
	CaptureRequestBody  bool
	CaptureResponseBody bool
	MaxBodyLength       int
	IncludePrefixes     []string
	ExcludePatterns     []string
	MaskedFields        []string
}

type StorageConfig struct {
	BasePath      string
	MaxUploadSize int64
}

type MailConfig struct {
	Enabled                      bool
	Host                         string
	Port                         int
	Username                     string
	Password                     string
	TLS                          bool
	From                         string
	FromName                     string
	AppURL                       string
	PasswordResetTTL             time.Duration
	PasswordResetCleanupInterval time.Duration
}

type BootstrapConfig struct {
	AdminUsername string
	AdminEmail    string
	AdminPassword string
}

type MigrationsConfig struct {
	Dir     string
	AutoRun bool
}

// Load reads .env if present, then resolves typed config from environment with
// in-code defaults. Returns an error only for unparseable values, not missing
// optional ones — every field has a sensible default for local dev.
func Load() (*Config, error) {
	_ = godotenv.Load() // .env is optional; ignore not-found

	cfg := &Config{
		App: AppConfig{
			Name: getString("APP_NAME", "gin-template"),
			Env:  getString("APP_ENV", "dev"),
		},
		Server: ServerConfig{
			Port:            getString("SERVER_PORT", "8080"),
			ReadTimeout:     getDuration("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getDuration("SERVER_WRITE_TIMEOUT", 15*time.Second),
			ShutdownTimeout: getDuration("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		DB: DBConfig{
			Driver:          getString("DB_DRIVER", "postgres"),
			Host:            getString("DB_HOST", "localhost"),
			Port:            getString("DB_PORT", "5432"),
			User:            getString("DB_USER", "postgres"),
			Password:        getString("DB_PASSWORD", "postgres"),
			Name:            getString("DB_NAME", "gin_template"),
			SSLMode:         getString("DB_SSLMODE", "disable"),
			TimeZone:        getString("DB_TIMEZONE", "UTC"),
			MaxOpenConns:    getInt("DB_POOL_MAX_OPEN", 20),
			MaxIdleConns:    getInt("DB_POOL_MAX_IDLE", 5),
			ConnMaxLifetime: getDuration("DB_POOL_CONN_MAX_LIFETIME", time.Hour),
		},
		Log: LogConfig{
			Level:      getString("LOG_LEVEL", "info"),
			Format:     getString("LOG_FORMAT", "console"),
			Dir:        getString("LOG_DIR", "logs"),
			File:       getString("LOG_FILE", "app.log"),
			MaxSizeMB:  getInt("LOG_MAX_SIZE_MB", 100),
			MaxBackups: getInt("LOG_MAX_BACKUPS", 7),
			MaxAgeDays: getInt("LOG_MAX_AGE_DAYS", 2),
			Compress:   getBool("LOG_COMPRESS", true),
		},
		JWT: JWTConfig{
			Secret:     getString("JWT_SECRET", "change-me-in-prod"),
			Issuer:     getString("JWT_ISSUER", "gin-template"),
			AccessTTL:  getDuration("JWT_ACCESS_TTL", 60*time.Minute),
			RefreshTTL: getDuration("JWT_REFRESH_TTL", 14*24*time.Hour),
		},
		Sessions: SessionsConfig{
			CleanupInterval:  getDuration("SESSIONS_CLEANUP_INTERVAL", time.Hour),
			CleanupRetention: getDuration("SESSIONS_CLEANUP_RETENTION", 7*24*time.Hour),
		},
		CORS: CORSConfig{
			AllowedOrigins:   getCSV("CORS_ALLOWED_ORIGINS", []string{"http://localhost:*", "http://127.0.0.1:*"}),
			AllowedMethods:   getCSV("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
			AllowedHeaders:   getCSV("CORS_ALLOWED_HEADERS", []string{"*"}),
			ExposedHeaders:   getCSV("CORS_EXPOSED_HEADERS", []string{"Authorization", "X-Request-Id", "X-RateLimit-Limit", "X-RateLimit-Remaining", "Retry-After"}),
			AllowCredentials: getBool("CORS_ALLOW_CREDENTIALS", true),
			MaxAge:           time.Duration(getInt("CORS_MAX_AGE", 3600)) * time.Second,
		},
		RateLimit: RateLimitConfig{
			Enabled:          getBool("RATE_LIMIT_ENABLED", true),
			Capacity:         getInt("RATE_LIMIT_CAPACITY", 60),
			RefillTokens:     getInt("RATE_LIMIT_REFILL_TOKENS", 60),
			RefillPeriod:     getDuration("RATE_LIMIT_REFILL_PERIOD", time.Minute),
			AuthCapacity:     getInt("RATE_LIMIT_AUTH_CAPACITY", 10),
			AuthRefillTokens: getInt("RATE_LIMIT_AUTH_REFILL_TOKENS", 10),
			AuthRefillPeriod: getDuration("RATE_LIMIT_AUTH_REFILL_PERIOD", time.Minute),
			AuthPathPrefixes: getCSV("RATE_LIMIT_AUTH_PATH_PREFIXES", []string{"/api/v1/auth/"}),
			IncludePrefixes:  getCSV("RATE_LIMIT_INCLUDE_PATH_PREFIXES", []string{"/api/"}),
		},
		Idem: IdempotencyConfig{
			Enabled:         getBool("IDEMPOTENCY_ENABLED", true),
			TTL:             getDuration("IDEMPOTENCY_TTL", 24*time.Hour),
			CleanupInterval: getDuration("IDEMPOTENCY_CLEANUP_INTERVAL", time.Hour),
		},
		Audit: AuditConfig{
			Enabled:             getBool("AUDIT_ENABLED", true),
			ExposeAPI:           getBool("AUDIT_EXPOSE_API", true),
			AuditAll:            getBool("AUDIT_AUDIT_ALL", true),
			CaptureRequestBody:  getBool("AUDIT_CAPTURE_REQUEST_BODY", true),
			CaptureResponseBody: getBool("AUDIT_CAPTURE_RESPONSE_BODY", true),
			MaxBodyLength:       getInt("AUDIT_MAX_BODY_LENGTH", 10000),
			IncludePrefixes:     getCSV("AUDIT_INCLUDE_PATH_PREFIXES", []string{"/api/"}),
			ExcludePatterns:     getCSV("AUDIT_EXCLUDE_PATH_PATTERNS", []string{"/swagger", "/api/v1/files/"}),
			MaskedFields:        getCSV("AUDIT_MASKED_FIELDS", []string{"password", "passwordHash", "currentPassword", "newPassword", "token", "accessToken", "refreshToken", "secret", "apiKey", "authorization"}),
		},
		Storage: StorageConfig{
			BasePath:      getString("STORAGE_BASE_PATH", "uploads"),
			MaxUploadSize: int64(getInt("STORAGE_MAX_UPLOAD_SIZE", 10*1024*1024)),
		},
		Mail: MailConfig{
			Enabled:                      getBool("MAIL_ENABLED", false),
			Host:                         getString("MAIL_HOST", "smtp.example.com"),
			Port:                         getInt("MAIL_PORT", 587),
			Username:                     getString("MAIL_USERNAME", ""),
			Password:                     getString("MAIL_PASSWORD", ""),
			TLS:                          getBool("MAIL_TLS", true),
			From:                         getString("MAIL_FROM", "no-reply@example.com"),
			FromName:                     getString("MAIL_FROM_NAME", "Gin Template"),
			AppURL:                       getString("MAIL_APP_URL", "http://localhost:8080"),
			PasswordResetTTL:             getDuration("MAIL_PASSWORD_RESET_TTL", 30*time.Minute),
			PasswordResetCleanupInterval: getDuration("MAIL_PASSWORD_RESET_CLEANUP_INTERVAL", time.Hour),
		},
		Bootstrap: BootstrapConfig{
			AdminUsername: getString("BOOTSTRAP_ADMIN_USERNAME", "admin"),
			AdminEmail:    getString("BOOTSTRAP_ADMIN_EMAIL", "admin@gmail.com"),
			AdminPassword: getString("BOOTSTRAP_ADMIN_PASSWORD", "admin123"),
		},
		Migrations: MigrationsConfig{
			Dir:     getString("MIGRATIONS_DIR", "migrations"),
			AutoRun: getBool("MIGRATIONS_AUTO_RUN", true),
		},
	}

	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if cfg.App.Env == "prod" && strings.HasPrefix(cfg.JWT.Secret, "change-me") {
		return nil, fmt.Errorf("JWT_SECRET must be overridden in prod")
	}

	return cfg, nil
}

func getString(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getBool(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func getDuration(key string, def time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func getCSV(key string, def []string) []string {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return def
	}
	return out
}
