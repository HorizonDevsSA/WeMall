package config

import "os"

// Config holds all runtime configuration for the ecocash-service.
// Sensitive fields (MerchantPin, Password) are intentionally not logged —
// callers must redact them before writing to any logger.
type Config struct {
	GRPCPort    string
	Environment string
	DBURL       string
	NatsURL     string

	// EcoCash EIP credentials
	EcoCashBaseURL        string // https://developers.ecocash.co.zw/sandbox/payment/v1
	EcoCashUsername       string
	EcoCashPassword       string // used for Basic Auth — never log
	EcoCashMerchantCode   string
	EcoCashMerchantPin    string // never log
	EcoCashMerchantNumber string
	EcoCashTerminalID     string
	EcoCashCountryCode    string
	EcoCashMerchantName   string
	EcoCashSuperMerchant  string

	// The public URL the service is reachable at — embedded in notifyUrl
	// e.g. https://api.wemall.co.zw/webhooks/ecocash or an ngrok URL in dev
	WebhookBaseURL string
	EcoCashProxySecret string
}

// Load reads configuration from environment variables with safe defaults for
// development. Production values must be injected via env / secrets manager.
func Load() (*Config, error) {
	return &Config{
		GRPCPort:    getEnv("GRPC_PORT", "9018"),
		Environment: getEnv("ENVIRONMENT", "development"),
		DBURL:       getEnv("DB_URL", "postgres://wemall:wemall_secret@localhost:5446/wemall_ecocash?sslmode=disable"),
		NatsURL:     getEnv("NATS_URL", "nats://localhost:4222"),

		EcoCashBaseURL:        getEnv("ECOCASH_BASE_URL", "https://developers.ecocash.co.zw/sandbox/payment/v1"),
		EcoCashUsername:       getEnv("ECOCASH_USERNAME", ""),
		EcoCashPassword:       getEnv("ECOCASH_PASSWORD", ""),
		EcoCashMerchantCode:   getEnv("ECOCASH_MERCHANT_CODE", ""),
		EcoCashMerchantPin:    getEnv("ECOCASH_MERCHANT_PIN", ""),
		EcoCashMerchantNumber: getEnv("ECOCASH_MERCHANT_NUMBER", ""),
		EcoCashTerminalID:     getEnv("ECOCASH_TERMINAL_ID", ""),
		EcoCashCountryCode:    getEnv("ECOCASH_COUNTRY_CODE", "ZW"),
		EcoCashMerchantName:   getEnv("ECOCASH_MERCHANT_NAME", "WeMall"),
		EcoCashSuperMerchant:  getEnv("ECOCASH_SUPER_MERCHANT", ""),

		WebhookBaseURL:     getEnv("WEBHOOK_BASE_URL", "http://localhost:8080"),
		EcoCashProxySecret: getEnv("ECOCASH_PROXY_SECRET", ""),
	}, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
