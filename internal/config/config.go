package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr   string
	MySQLDSN   string
	JWTSecret  string
	AdminUser  string
	AdminPass  string

	PayPalClientID  string
	PayPalSecret    string
	PayPalSandbox   bool
	PayPalWebhookID string

	TaxJarAPIKey string

	DownloadSecret  string
	DownloadTTL     time.Duration
	DownloadBaseURL string

	FileStoragePath string
}

func Load() *Config {
	ttl, err := time.ParseDuration(getEnv("DOWNLOAD_TTL", "24h"))
	if err != nil {
		log.Fatalf("invalid DOWNLOAD_TTL: %v", err)
	}
	sandbox, _ := strconv.ParseBool(getEnv("PAYPAL_SANDBOX", "true"))

	return &Config{
		HTTPAddr:        getEnv("HTTP_ADDR", ":8080"),
		MySQLDSN:        getEnv("MYSQL_DSN", ""),
		JWTSecret:       getEnv("JWT_SECRET", "dev-secret"),
		AdminUser:       getEnv("ADMIN_USERNAME", "admin"),
		AdminPass:       getEnv("ADMIN_PASSWORD", "admin123"),

		PayPalClientID:  getEnv("PAYPAL_CLIENT_ID", ""),
		PayPalSecret:    getEnv("PAYPAL_SECRET", ""),
		PayPalSandbox:   sandbox,
		PayPalWebhookID: getEnv("PAYPAL_WEBHOOK_ID", ""),

		TaxJarAPIKey: getEnv("TAXJAR_API_KEY", ""),

		DownloadSecret:  getEnv("DOWNLOAD_SECRET", "dev-download"),
		DownloadTTL:     ttl,
		DownloadBaseURL: getEnv("DOWNLOAD_BASE_URL", "http://localhost:8080"),

		FileStoragePath: getEnv("FILE_STORAGE_PATH", "./storage"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
