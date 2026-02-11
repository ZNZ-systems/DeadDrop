package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port        int
	DatabaseURL string

	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPass     string
	SMTPFrom     string
	SMTPEnabled  bool

	RateLimitRPS   float64
	RateLimitBurst int

	SessionMaxAge  int // hours
	BaseURL        string
	SecureCookies  bool
}

func Load() (*Config, error) {
	port, err := getIntEnv("PORT", 8080)
	if err != nil {
		return nil, fmt.Errorf("invalid PORT: %w", err)
	}

	dbURL := getEnv("DATABASE_URL", "postgres://deaddrop:deaddrop@localhost:5432/deaddrop?sslmode=disable")

	smtpPort, err := getIntEnv("SMTP_PORT", 587)
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP_PORT: %w", err)
	}

	rps, err := getFloatEnv("RATE_LIMIT_RPS", 2.0)
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_RPS: %w", err)
	}

	burst, err := getIntEnv("RATE_LIMIT_BURST", 5)
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_BURST: %w", err)
	}

	sessionMaxAge, err := getIntEnv("SESSION_MAX_AGE_HOURS", 72)
	if err != nil {
		return nil, fmt.Errorf("invalid SESSION_MAX_AGE_HOURS: %w", err)
	}

	smtpHost := getEnv("SMTP_HOST", "")

	return &Config{
		Port:           port,
		DatabaseURL:    dbURL,
		SMTPHost:       smtpHost,
		SMTPPort:       smtpPort,
		SMTPUser:       getEnv("SMTP_USER", ""),
		SMTPPass:       getEnv("SMTP_PASS", ""),
		SMTPFrom:       getEnv("SMTP_FROM", ""),
		SMTPEnabled:    smtpHost != "",
		RateLimitRPS:   rps,
		RateLimitBurst: burst,
		SessionMaxAge:  sessionMaxAge,
		BaseURL:        getEnv("BASE_URL", "http://localhost:8080"),
		SecureCookies:  getEnv("SECURE_COOKIES", "true") != "false",
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getIntEnv(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	return strconv.Atoi(v)
}

func getFloatEnv(key string, fallback float64) (float64, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	return strconv.ParseFloat(v, 64)
}
