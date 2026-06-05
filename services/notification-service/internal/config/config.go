package config

import (
	"os"
)

type Config struct {
	Environment        string
	GRPCPort           string
	DBURL              string
	RedisURL           string
	NatsURL            string
	SellerServiceAddr  string
	SMTPHost           string
	SMTPPort           string
	SMTPUser           string
	SMTPPass           string
	FirebaseCredPath   string // Optional file path to Firebase Service Account JSON
	FirebaseCredJSON   string // Optional raw JSON string
}

func Load() (*Config, error) {
	return &Config{
		Environment:        getEnv("ENVIRONMENT", "development"),
		GRPCPort:           getEnv("GRPC_PORT", "9007"),
		DBURL:              getEnv("DB_URL", "postgres://wemall:wemall_secret@localhost:5436/wemall_notifications?sslmode=disable"),
		RedisURL:           getEnv("REDIS_URL", "localhost:6379"),
		NatsURL:            getEnv("NATS_URL", "nats://localhost:4222"),
		SellerServiceAddr:  getEnv("SELLER_SERVICE_ADDR", "localhost:9002"),
		SMTPHost:           getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:           getEnv("SMTP_PORT", "587"),
		SMTPUser:           getEnv("SMTP_USER", ""),
		SMTPPass:           getEnv("SMTP_PASS", ""),
		FirebaseCredPath:   getEnv("FIREBASE_CREDENTIALS_PATH", ""),
		FirebaseCredJSON:   getEnv("FIREBASE_CREDENTIALS_JSON", ""),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
