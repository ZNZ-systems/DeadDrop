package blob

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrObjectNotFound = errors.New("blob object not found")

type Store interface {
	Put(ctx context.Context, key, contentType string, body []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

type Config struct {
	Backend           string
	FSRoot            string
	S3Bucket          string
	S3Region          string
	S3Endpoint        string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3ForcePathStyle  bool
}

func NewFromConfig(ctx context.Context, cfg Config) (Store, error) {
	backend := strings.ToLower(strings.TrimSpace(cfg.Backend))
	if backend == "" {
		backend = "filesystem"
	}

	switch backend {
	case "filesystem", "fs", "local":
		return NewFilesystemStore(cfg.FSRoot)
	case "s3", "r2":
		return NewS3Store(ctx, S3Config{
			Bucket:          cfg.S3Bucket,
			Region:          cfg.S3Region,
			Endpoint:        cfg.S3Endpoint,
			AccessKeyID:     cfg.S3AccessKeyID,
			SecretAccessKey: cfg.S3SecretAccessKey,
			ForcePathStyle:  cfg.S3ForcePathStyle,
		})
	default:
		return nil, fmt.Errorf("unsupported blob backend: %s", backend)
	}
}
