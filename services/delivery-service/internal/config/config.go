package config

import "os"

type Config struct {
	GRPCPort           string
	Environment        string
	DBURL              string
	NatsURL            string
	RedisURL           string
	PaymentServiceAddr string
	UserServiceAddr    string
	SellerServiceAddr  string
}

func Load() (*Config, error) {
	return &Config{
		GRPCPort:           getEnv("GRPC_PORT", "9017"),
		Environment:        getEnv("ENVIRONMENT", "development"),
		DBURL:              getEnv("DB_URL", "postgres://wemall:wemall_secret@localhost:5445/wemall_delivery?sslmode=disable"),
		NatsURL:            getEnv("NATS_URL", "nats://localhost:4222"),
		RedisURL:           getEnv("REDIS_URL", "localhost:6379"),
		PaymentServiceAddr: getEnv("PAYMENT_SERVICE_ADDR", "localhost:9011"),
		UserServiceAddr:    getEnv("USER_SERVICE_ADDR", "localhost:9001"),
		SellerServiceAddr:  getEnv("SELLER_SERVICE_ADDR", "localhost:9002"),
	}, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
