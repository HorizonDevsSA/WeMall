package config

import "os"

// Config holds review-service configuration.
type Config struct {
	GRPCPort         string
	Environment      string
	DBURL            string
	NatsURL          string
	OrderServiceAddr string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	return &Config{
		GRPCPort:         getEnv("GRPC_PORT", "9009"), // Using port 9009 for review-service
		Environment:      getEnv("ENVIRONMENT", "development"),
		DBURL:            getEnv("DB_URL", "postgres://wemall:wemall_secret@localhost:5438/wemall_reviews?sslmode=disable"),
		NatsURL:          getEnv("NATS_URL", "nats://localhost:4222"),
		OrderServiceAddr: getEnv("ORDER_SERVICE_ADDR", "localhost:9005"),
	}, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
