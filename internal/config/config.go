package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	AI       AIConfig
	Security SecurityConfig
	Logging  LoggingConfig
}

type ServerConfig struct {
	Port         string
	Host         string
	ReadTimeout  int
	WriteTimeout int
}

type DatabaseConfig struct {
	MongoURI    string
	Database    string
	MaxPoolSize int
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type AIConfig struct {
	LLMProvider    string
	LLMAPIKey      string
	EmbeddingModel string
}

type SecurityConfig struct {
	JWTSecret     string
	EncryptionKey string
	TLSCertPath   string
	TLSKeyPath    string
}

type LoggingConfig struct {
	Level  string
	Format string
	Output string
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			ReadTimeout:  getEnvAsInt("SERVER_READ_TIMEOUT", 30),
			WriteTimeout: getEnvAsInt("SERVER_WRITE_TIMEOUT", 30),
		},
		Database: DatabaseConfig{
			MongoURI:    getEnv("MONGO_URI", "mongodb://localhost:27017"),
			Database:    getEnv("MONGO_DATABASE", "ai_government_consultant"),
			MaxPoolSize: getEnvAsInt("MONGO_MAX_POOL_SIZE", 100),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		AI: AIConfig{
			LLMProvider:    getEnv("LLM_PROVIDER", "gemini"),
			LLMAPIKey:      getEnv("LLM_API_KEY", ""),
			EmbeddingModel: getEnv("EMBEDDING_MODEL", "text-embedding-004"),
		},
		Security: SecurityConfig{
			JWTSecret:     getEnv("JWT_SECRET", ""),
			EncryptionKey: getEnv("ENCRYPTION_KEY", ""),
			TLSCertPath:   getEnv("TLS_CERT_PATH", ""),
			TLSKeyPath:    getEnv("TLS_KEY_PATH", ""),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
			Output: getEnv("LOG_OUTPUT", "stdout"),
		},
	}

	// Validate required configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Security.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if c.AI.LLMAPIKey == "" {
		return fmt.Errorf("LLM_API_KEY is required")
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
