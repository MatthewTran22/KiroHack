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
	Research ResearchConfig
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

type ResearchConfig struct {
	NewsAPIKey          string
	NewsAPIBaseURL      string
	LLMModel            string
	MaxConcurrentRequests int
	RequestTimeout      int
	CacheEnabled        bool
	CacheTTL            int
	DefaultLanguage     string
	MaxSourcesPerQuery  int
	MinCredibilityScore float64
	MinRelevanceScore   float64
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
		Research: ResearchConfig{
			NewsAPIKey:            getEnv("NEWS_API_KEY", ""),
			NewsAPIBaseURL:        getEnv("NEWS_API_BASE_URL", "https://newsapi.org/v2"),
			LLMModel:              getEnv("RESEARCH_LLM_MODEL", "gemini-1.5-flash"),
			MaxConcurrentRequests: getEnvAsInt("RESEARCH_MAX_CONCURRENT_REQUESTS", 5),
			RequestTimeout:        getEnvAsInt("RESEARCH_REQUEST_TIMEOUT", 30),
			CacheEnabled:          getEnvAsBool("RESEARCH_CACHE_ENABLED", true),
			CacheTTL:              getEnvAsInt("RESEARCH_CACHE_TTL", 3600),
			DefaultLanguage:       getEnv("RESEARCH_DEFAULT_LANGUAGE", "en"),
			MaxSourcesPerQuery:    getEnvAsInt("RESEARCH_MAX_SOURCES_PER_QUERY", 20),
			MinCredibilityScore:   getEnvAsFloat("RESEARCH_MIN_CREDIBILITY_SCORE", 0.6),
			MinRelevanceScore:     getEnvAsFloat("RESEARCH_MIN_RELEVANCE_SCORE", 0.5),
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

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}
