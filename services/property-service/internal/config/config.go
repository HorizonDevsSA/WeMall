package config

import "os"

// Config holds property-service configuration.
type Config struct {
	GRPCPort    string
	Environment string
	DBURL       string
	NatsURL     string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	return &Config{
		GRPCPort:    getEnv("GRPC_PORT", "9019"),
		Environment: getEnv("ENVIRONMENT", "development"),
		DBURL:       getEnv("DB_URL", "postgres://wemall:wemall_secret@localhost:5447/wemall_property?sslmode=disable"),
		NatsURL:     getEnv("NATS_URL", "nats://localhost:4222"),
	}, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
