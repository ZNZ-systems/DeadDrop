package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port        int
	DatabaseURL string

	SMTPHost    string
	SMTPPort    int
	SMTPUser    string
	SMTPPass    string
	SMTPFrom    string
	SMTPEnabled bool

	RateLimitRPS           float64
	RateLimitBurst         int
	APIMaxBodyBytes        int64
	InboundAPIToken        string
	InboundAPIMaxBodyBytes int64
	InboundJobMaxAttempts  int
	InboundWorkerPollMS    int
	InboundMXTarget        string

	BlobBackend           string
	BlobFSRoot            string
	BlobS3Bucket          string
	BlobS3Region          string
	BlobS3Endpoint        string
	BlobS3AccessKeyID     string
	BlobS3SecretAccessKey string
	BlobS3ForcePathStyle  bool

	SessionMaxAge int // hours
	BaseURL       string
	SecureCookies bool
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

	apiMaxBodyBytes, err := getInt64Env("API_MAX_BODY_BYTES", 16*1024)
	if err != nil {
		return nil, fmt.Errorf("invalid API_MAX_BODY_BYTES: %w", err)
	}
	inboundAPIMaxBodyBytes, err := getInt64Env("INBOUND_API_MAX_BODY_BYTES", 1024*1024)
	if err != nil {
		return nil, fmt.Errorf("invalid INBOUND_API_MAX_BODY_BYTES: %w", err)
	}
	inboundJobMaxAttempts, err := getIntEnv("INBOUND_JOB_MAX_ATTEMPTS", 5)
	if err != nil {
		return nil, fmt.Errorf("invalid INBOUND_JOB_MAX_ATTEMPTS: %w", err)
	}
	inboundWorkerPollMS, err := getIntEnv("INBOUND_WORKER_POLL_MS", 500)
	if err != nil {
		return nil, fmt.Errorf("invalid INBOUND_WORKER_POLL_MS: %w", err)
	}

	sessionMaxAge, err := getIntEnv("SESSION_MAX_AGE_HOURS", 72)
	if err != nil {
		return nil, fmt.Errorf("invalid SESSION_MAX_AGE_HOURS: %w", err)
	}

	smtpHost := getEnv("SMTP_HOST", "")

	return &Config{
		Port:                   port,
		DatabaseURL:            dbURL,
		SMTPHost:               smtpHost,
		SMTPPort:               smtpPort,
		SMTPUser:               getEnv("SMTP_USER", ""),
		SMTPPass:               getEnv("SMTP_PASS", ""),
		SMTPFrom:               getEnv("SMTP_FROM", ""),
		SMTPEnabled:            smtpHost != "",
		RateLimitRPS:           rps,
		RateLimitBurst:         burst,
		APIMaxBodyBytes:        apiMaxBodyBytes,
		InboundAPIToken:        getEnv("INBOUND_API_TOKEN", ""),
		InboundAPIMaxBodyBytes: inboundAPIMaxBodyBytes,
		InboundJobMaxAttempts:  inboundJobMaxAttempts,
		InboundWorkerPollMS:    inboundWorkerPollMS,
		InboundMXTarget:        getEnv("INBOUND_MX_TARGET", "mx.deaddrop.local"),
		BlobBackend:            getEnv("BLOB_BACKEND", "filesystem"),
		BlobFSRoot:             getEnv("BLOB_FS_ROOT", "./data/blobs"),
		BlobS3Bucket:           getEnv("BLOB_S3_BUCKET", ""),
		BlobS3Region:           getEnv("BLOB_S3_REGION", "auto"),
		BlobS3Endpoint:         getEnv("BLOB_S3_ENDPOINT", ""),
		BlobS3AccessKeyID:      getEnv("BLOB_S3_ACCESS_KEY_ID", ""),
		BlobS3SecretAccessKey:  getEnv("BLOB_S3_SECRET_ACCESS_KEY", ""),
		BlobS3ForcePathStyle:   getBoolEnv("BLOB_S3_FORCE_PATH_STYLE", false),
		SessionMaxAge:          sessionMaxAge,
		BaseURL:                getEnv("BASE_URL", "http://localhost:8080"),
		SecureCookies:          getEnv("SECURE_COOKIES", "true") != "false",
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

func getInt64Env(key string, fallback int64) (int64, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	return strconv.ParseInt(v, 10, 64)
}

func getBoolEnv(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return parsed
}
