package config

import "os"

type Config struct {
	DB_URL            string
	UsersDBURL        string
	SellersDBURL      string
	DisputesDBURL     string
	OrdersDBURL       string
	GRPCPort          string
	SellerServiceAddr string
	Environment       string
}

func Load() *Config {
	return &Config{
		DB_URL:            getEnv("DB_URL", "postgres://wemall:wemall_secret@localhost:5442/wemall_admin?sslmode=disable"),
		UsersDBURL:        getEnv("USERS_DB_URL", "postgres://wemall:wemall_secret@localhost:5432/wemall_users?sslmode=disable"),
		SellersDBURL:      getEnv("SELLERS_DB_URL", "postgres://wemall:wemall_secret@localhost:5435/wemall_sellers?sslmode=disable"),
		DisputesDBURL:     getEnv("DISPUTES_DB_URL", "postgres://wemall:wemall_secret@localhost:5441/wemall_dispute?sslmode=disable"),
		OrdersDBURL:       getEnv("ORDERS_DB_URL", "postgres://wemall:wemall_secret@localhost:5434/wemall_orders?sslmode=disable"),
		GRPCPort:          getEnv("GRPC_PORT", "9014"),
		SellerServiceAddr: getEnv("SELLER_SERVICE_ADDR", "localhost:9002"),
		Environment:       getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
