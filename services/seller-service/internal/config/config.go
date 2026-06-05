package config

import "os"

// Config holds seller-service configuration.
type Config struct {
	GRPCPort    string
	Environment string
	DBURL       string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	return &Config{
		GRPCPort:    getEnv("GRPC_PORT", "9002"),
		Environment: getEnv("ENVIRONMENT", "development"),
		DBURL:       getEnv("DB_URL", "postgres://wemall:wemall_secret@localhost:5435/wemall_sellers?sslmode=disable"),
	}, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
