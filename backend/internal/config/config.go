package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config holds all application configuration
type Config struct {
	App      AppConfig
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Cloud    CloudConfig
	Slack    SlackConfig
	AI       AIConfig
	Logging  LoggingConfig
	Email    EmailConfig
}

// AppConfig holds application-level configuration
type AppConfig struct {
	Env      string // development, staging, production
	DemoMode bool   // Use mock data when true
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         string
	Mode         string // debug, release, test
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

// AIConfig holds AI service configuration
type AIConfig struct {
	Enabled bool
	BaseURL string
	Model   string
	Timeout time.Duration
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// JWTConfig holds JWT token configuration
type JWTConfig struct {
	Secret      string
	ExpireHours int
}

// CloudConfig holds cloud provider API configuration
type CloudConfig struct {
	AWS   AWSConfig
	GCP   GCPConfig
	Azure AzureConfig
}

// AWSConfig holds AWS-specific configuration
type AWSConfig struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
}

// GCPConfig holds GCP-specific configuration
type GCPConfig struct {
	ProjectID       string
	CredentialsFile string
}

// AzureConfig holds Azure-specific configuration
type AzureConfig struct {
	SubscriptionID string
	ClientID       string
	ClientSecret   string
	TenantID       string
}

// SlackConfig holds Slack integration configuration
type SlackConfig struct {
	BotToken      string
	SigningSecret string
	WebhookURL    string
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level     string
	SentryDSN string
}

// EmailConfig holds email configuration for alerts
type EmailConfig struct {
	SMTPHost string
	SMTPPort int
	User     string
	Password string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found, using environment variables")
	}

	config := &Config{
		App: AppConfig{
			Env:      getEnv("APP_ENV", "development"),
			DemoMode: getBoolEnv("DEMO_MODE", false),
		},
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			Mode:         getEnv("GIN_MODE", "debug"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			Name:     getEnv("DB_NAME", "devcost_ai"),
			User:     getEnv("DB_USER", "devcost"),
			Password: getEnv("DB_PASSWORD", ""),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getIntEnv("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:      getEnv("JWT_SECRET", ""),
			ExpireHours: getIntEnv("JWT_EXPIRE_HOURS", 24),
		},
		Cloud: CloudConfig{
			AWS: AWSConfig{
				AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
				SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
				Region:          getEnv("AWS_REGION", "us-east-1"),
			},
			GCP: GCPConfig{
				ProjectID:       getEnv("GCP_PROJECT_ID", ""),
				CredentialsFile: getEnv("GCP_CREDENTIALS_FILE", ""),
			},
			Azure: AzureConfig{
				SubscriptionID: getEnv("AZURE_SUBSCRIPTION_ID", ""),
				ClientID:       getEnv("AZURE_CLIENT_ID", ""),
				ClientSecret:   getEnv("AZURE_CLIENT_SECRET", ""),
				TenantID:       getEnv("AZURE_TENANT_ID", ""),
			},
		},
		Slack: SlackConfig{
			BotToken:      getEnv("SLACK_BOT_TOKEN", ""),
			SigningSecret: getEnv("SLACK_SIGNING_SECRET", ""),
			WebhookURL:    getEnv("SLACK_WEBHOOK_URL", ""),
		},
		AI: AIConfig{
			Enabled: getBoolEnv("AI_ENABLED", false),
			BaseURL: getEnv("AI_BASE_URL", "http://localhost:11434"),
			Model:   getEnv("AI_MODEL", "llama3.2"),
			Timeout: getDurationEnv("AI_TIMEOUT", 30*time.Second),
		},
		Logging: LoggingConfig{
			Level:     getEnv("LOG_LEVEL", "info"),
			SentryDSN: getEnv("SENTRY_DSN", ""),
		},
		Email: EmailConfig{
			SMTPHost: getEnv("SMTP_HOST", ""),
			SMTPPort: getIntEnv("SMTP_PORT", 587),
			User:     getEnv("SMTP_USER", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
		},
	}

	// Validate required configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	var errors []string

	// Server validation
	if c.Server.Port == "" {
		errors = append(errors, "SERVER_PORT is required")
	}

	// Database validation (always required)
	if c.Database.Host == "" {
		errors = append(errors, "DB_HOST is required")
	}
	if c.Database.Name == "" {
		errors = append(errors, "DB_NAME is required")
	}
	if c.Database.User == "" {
		errors = append(errors, "DB_USER is required")
	}
	if c.Database.Password == "" {
		errors = append(errors, "DB_PASSWORD is required")
	}

	// JWT validation for production
	if c.App.Env == "production" {
		if c.JWT.Secret == "" {
			errors = append(errors, "JWT_SECRET is required in production")
		}
	}

	// AWS validation (only if not in demo mode)
	if !c.App.DemoMode {
		if c.Cloud.AWS.AccessKeyID == "" {
			errors = append(errors, "AWS_ACCESS_KEY_ID is required when DEMO_MODE=false")
		}
		if c.Cloud.AWS.SecretAccessKey == "" {
			errors = append(errors, "AWS_SECRET_ACCESS_KEY is required when DEMO_MODE=false")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	// Warnings
	if c.App.DemoMode {
		fmt.Println("INFO: Running in DEMO_MODE - using mock data")
	}
	if c.JWT.Secret == "" && c.App.Env != "production" {
		fmt.Println("WARNING: JWT_SECRET not set - using insecure default for development")
		c.JWT.Secret = "dev-secret-do-not-use-in-production"
	}

	return nil
}

// IsDemoMode returns true if the app is running in demo mode
func (c *Config) IsDemoMode() bool {
	return c.App.DemoMode
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}

// HasAWSCredentials returns true if AWS credentials are configured
func (c *Config) HasAWSCredentials() bool {
	return c.Cloud.AWS.AccessKeyID != "" && c.Cloud.AWS.SecretAccessKey != ""
}

// HasSlackConfig returns true if Slack is configured
func (c *Config) HasSlackConfig() bool {
	return c.Slack.BotToken != "" || c.Slack.WebhookURL != ""
}

// DatabaseURL returns the database connection URL
func (c *DatabaseConfig) DatabaseURL() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}

// RedisURL returns the Redis connection URL
func (c *RedisConfig) RedisURL() string {
	if c.Password != "" {
		return fmt.Sprintf("%s:%s@%s:%s/%d", "", c.Password, c.Host, c.Port, c.DB)
	}
	return fmt.Sprintf("%s:%s/%d", c.Host, c.Port, c.DB)
}

// ToZapConfig converts logging config to Zap config
func (c *LoggingConfig) ToZapConfig() (*zap.Config, error) {
	var level zap.AtomicLevel
	switch c.Level {
	case "debug":
		level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	zapConfig := zap.NewProductionConfig()
	zapConfig.Level = level
	zapConfig.OutputPaths = []string{"stdout"}
	zapConfig.ErrorOutputPaths = []string{"stderr"}
	zapConfig.EncoderConfig.TimeKey = "timestamp"
	zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapConfig.EncoderConfig.StacktraceKey = ""

	return &zapConfig, nil
}

// Helper functions for environment variable parsing
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
