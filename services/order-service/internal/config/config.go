package config

import "os"

type Config struct {
	GRPCPort           string
	Environment        string
	DBURL              string
	NatsURL            string
	ProductServiceAddr string
	SellerServiceAddr  string
}

func Load() (*Config, error) {
	return &Config{
		GRPCPort:           getEnv("GRPC_PORT", "9005"),
		Environment:        getEnv("ENVIRONMENT", "development"),
		DBURL:              getEnv("DB_URL", "postgres://wemall:wemall_secret@localhost:5434/wemall_orders?sslmode=disable"),
		NatsURL:            getEnv("NATS_URL", "nats://localhost:4222"),
		ProductServiceAddr: getEnv("PRODUCT_SERVICE_ADDR", "localhost:9003"),
		SellerServiceAddr:  getEnv("SELLER_SERVICE_ADDR", "localhost:9002"),
	}, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
