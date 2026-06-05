package config

import "os"

type Config struct {
	GRPCPort             string
	Environment          string
	DBURL                string
	SellerServiceAddr    string
}

func Load() (*Config, error) {
	return &Config{
		GRPCPort:             getEnv("GRPC_PORT", "9003"),
		Environment:          getEnv("ENVIRONMENT", "development"),
		DBURL:                getEnv("DB_URL", "postgres://wemall:wemall_secret@localhost:5433/wemall_products?sslmode=disable"),
		SellerServiceAddr:    getEnv("SELLER_SERVICE_ADDR", "localhost:9002"),
	}, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
