package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port              string
	DatabaseURL       string
	JWTSecret         string
	JWTExpiry         time.Duration
	RefreshExpiry     time.Duration
	RateLimitRequests int
	RateLimitWindow   time.Duration
	SMTPHost          string
	SMTPPort          int
	SMTPUsername      string
	SMTPPassword      string
	SMTPFrom          string
	AWSBucket         string
	AWSRegion         string
	AppURL            string
}

func Load() *Config {
	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/bus_booking?sslmode=disable"),
		JWTSecret:         getEnv("JWT_SECRET", "change-me-in-production"),
		JWTExpiry:         getDuration("JWT_EXPIRY", 15*time.Minute),
		RefreshExpiry:     getDuration("REFRESH_EXPIRY", 7*24*time.Hour),
		RateLimitRequests: getInt("RATE_LIMIT_REQUESTS", 20),
		RateLimitWindow:   getDuration("RATE_LIMIT_WINDOW", 1*time.Minute),
		SMTPHost:          getEnv("SMTP_HOST", "smtp.mailtrap.io"),
		SMTPPort:          getInt("SMTP_PORT", 587),
		SMTPUsername:      getEnv("SMTP_USERNAME", ""),
		SMTPPassword:      getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:          getEnv("SMTP_FROM", "noreply@busbooking.com"),
		AWSBucket:         getEnv("AWS_BUCKET", ""),
		AWSRegion:         getEnv("AWS_REGION", "us-east-1"),
		AppURL:            getEnv("APP_URL", "http://localhost:5173"),
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return fallback
}
