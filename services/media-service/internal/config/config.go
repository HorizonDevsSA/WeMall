package config

import "os"

type Config struct {
	HTTPPort             string
	GRPCPort             string
	Environment          string
	DBURL                string
	AWSRegion            string
	AWSS3RawBucket       string
	AWSS3PublicBucket    string
	AWSS3PrivateBucket   string
	AWSCloudFrontPublic  string
	AWSCloudFrontPrivate string
	AWSSecretsCFKeyName  string // Secrets Manager secret name for CloudFront Private Key
	CloudFrontKeyPairID  string // CloudFront Key-Pair-Id for signing private URLs
}

func Load() (*Config, error) {
	return &Config{
		HTTPPort:             getEnv("HTTP_PORT", "8087"),
		GRPCPort:             getEnv("GRPC_PORT", "50057"),
		Environment:          getEnv("ENVIRONMENT", "development"),
		DBURL:                getEnv("DB_URL", "postgres://postgres:postgres@localhost:5432/wemall_media?sslmode=disable"),
		AWSRegion:            getEnv("AWS_REGION", "us-east-1"),
		AWSS3RawBucket:       getEnv("AWS_S3_RAW_BUCKET", "wemall-media-raw"),
		AWSS3PublicBucket:    getEnv("AWS_S3_PUBLIC_BUCKET", "wemall-media-public"),
		AWSS3PrivateBucket:   getEnv("AWS_S3_PRIVATE_BUCKET", "wemall-media-private"),
		AWSCloudFrontPublic:  getEnv("AWS_CLOUDFRONT_PUBLIC_URL", "https://cdn.wemall.com"),
		AWSCloudFrontPrivate: getEnv("AWS_CLOUDFRONT_PRIVATE_URL", "https://private-cdn.wemall.com"),
		AWSSecretsCFKeyName:  getEnv("AWS_SECRETS_CF_KEY_NAME", "wemall/cloudfront/private-key"),
		CloudFrontKeyPairID:  getEnv("CLOUDFRONT_KEY_PAIR_ID", "K2JC486F2EXAMPLE"),
	}, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
