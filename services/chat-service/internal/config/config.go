package config

import (
	"os"
)

type Config struct {
	Port              string
	DatabaseURL       string
	NatsURL           string
	SellerServiceAddr string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("GRPC_PORT")
	}
	if port == "" {
		port = "50059"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DB_URL")
	}
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/wemall_chat?sslmode=disable"
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	sellerAddr := os.Getenv("SELLER_SERVICE_ADDR")
	if sellerAddr == "" {
		sellerAddr = "localhost:50052"
	}

	return &Config{
		Port:              port,
		DatabaseURL:       dbURL,
		NatsURL:           natsURL,
		SellerServiceAddr: sellerAddr,
	}
}
