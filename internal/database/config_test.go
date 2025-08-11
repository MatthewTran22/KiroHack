package database

import "time"

// DockerTestConfig returns a configuration for testing with Docker MongoDB
func DockerTestConfig() *Config {
	return &Config{
		URI:            "mongodb://admin:password@localhost:27017/ai_government_consultant?authSource=admin",
		DatabaseName:   "ai_government_consultant",
		ConnectTimeout: 10 * time.Second,
		MaxPoolSize:    10,
		MinPoolSize:    1,
		MaxIdleTime:    5 * time.Minute,
	}
}
