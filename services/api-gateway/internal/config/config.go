package config

import (
	"os"
)

type Config struct {
	Port                 string
	UserServiceAddr      string
	SellerServiceAddr    string
	ProductServiceAddr   string
	OrderServiceAddr     string
	InventoryServiceAddr string
	NotificationServiceAddr string
	ReviewServiceAddr    string
	PaymentServiceAddr   string
	AdminServiceAddr     string
	JWTSecret            string
	Environment          string
}

func Load() *Config {
	return &Config{
		Port:                 getEnv("PORT", "8080"),
		UserServiceAddr:      getEnv("USER_SERVICE_ADDR", "localhost:9001"),
		SellerServiceAddr:    getEnv("SELLER_SERVICE_ADDR", "localhost:9002"),
		ProductServiceAddr:   getEnv("PRODUCT_SERVICE_ADDR", "localhost:9003"),
		OrderServiceAddr:     getEnv("ORDER_SERVICE_ADDR", "localhost:9005"),
		InventoryServiceAddr: getEnv("INVENTORY_SERVICE_ADDR", "localhost:9006"),
		NotificationServiceAddr: getEnv("NOTIFICATION_SERVICE_ADDR", "localhost:9007"),
		ReviewServiceAddr:    getEnv("REVIEW_SERVICE_ADDR", "localhost:9009"),
		PaymentServiceAddr:   getEnv("PAYMENT_SERVICE_ADDR", "localhost:9011"),
		AdminServiceAddr:     getEnv("ADMIN_SERVICE_ADDR", "localhost:9014"),
		JWTSecret:            getEnv("JWT_SECRET", "super_secret_jwt_key_change_in_production"),
		Environment:          getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
