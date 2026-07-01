package config

import "os"

// Config holds seller-service configuration.
type Config struct {
	GRPCPort          string
	Environment       string
	DBURL             string
	NatsURL           string
	BankEncryptionKey string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	return &Config{
		GRPCPort:          getEnv("GRPC_PORT", "9002"),
		Environment:       getEnv("ENVIRONMENT", "development"),
		DBURL:             getEnv("DB_URL", "postgres://wemall:wemall_secret@localhost:5435/wemall_sellers?sslmode=disable"),
		NatsURL:           getEnv("NATS_URL", "nats://localhost:4222"),
		BankEncryptionKey: getEnv("SELLER_BANK_ENCRYPTION_KEY", "dev_placeholder_key_must_be_32_b"), // 32 bytes placeholder
	}, nil
}


func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
