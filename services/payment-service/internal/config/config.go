package config

import "os"

// Config holds payment-service configuration.
type Config struct {
	GRPCPort            string
	Environment         string
	DBURL               string
	NatsURL             string
	StripeSecretKey     string
	GooglePayMerchantID string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	return &Config{
		GRPCPort:            getEnv("GRPC_PORT", "9011"), // Using port 9011 for payment-service
		Environment:         getEnv("ENVIRONMENT", "development"),
		DBURL:               getEnv("DB_URL", "postgres://wemall:wemall_secret@localhost:5439/wemall_payments?sslmode=disable"),
		NatsURL:             getEnv("NATS_URL", "nats://localhost:4222"),
		StripeSecretKey:     getEnv("STRIPE_SECRET_KEY", "sk_test_mock_secret_key"),
		GooglePayMerchantID: getEnv("GOOGLE_PAY_MERCHANT_ID", "mock_merchant_id"),
	}, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
