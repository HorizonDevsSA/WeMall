package config

import (
	"os"
)

// Config holds all configuration for the user service.
type Config struct {
	GRPCPort    string
	Environment string
	DBUrl       string

	RedisURL string

	JWTSecret        string
	JWTRefreshSecret string

	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	AfricasTalkingAPIKey   string
	AfricasTalkingUsername string

	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
}

// Load reads config directly from environment variables.
func Load() (*Config, error) {
	return &Config{
		GRPCPort:    getEnv("GRPC_PORT", "9001"),
		Environment: getEnv("ENVIRONMENT", "development"),
		DBUrl:       getEnv("DB_URL", "postgres://wemall:wemall_secret@localhost:5432/wemall_users?sslmode=disable"),

		RedisURL: getEnv("REDIS_URL", "localhost:6379"),

		JWTSecret:        getEnv("JWT_SECRET", "super_secret_jwt_key_change_in_production"),
		JWTRefreshSecret: getEnv("JWT_REFRESH_SECRET", "super_secret_refresh_key_change_in_production"),

		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/google/callback"),

		AfricasTalkingAPIKey:   getEnv("AFRICAS_TALKING_API_KEY", ""),
		AfricasTalkingUsername: getEnv("AFRICAS_TALKING_USERNAME", "sandbox"),

		SMTPHost: getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort: getEnv("SMTP_PORT", "587"),
		SMTPUser: getEnv("SMTP_USER", ""),
		SMTPPass: getEnv("SMTP_PASS", ""),
	}, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
