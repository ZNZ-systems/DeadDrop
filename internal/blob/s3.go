package blob

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Config struct {
	Bucket          string
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	ForcePathStyle  bool
}

type S3Store struct {
	client *s3.Client
	bucket string
}

func NewS3Store(ctx context.Context, cfg S3Config) (*S3Store, error) {
	cfg.Bucket = strings.TrimSpace(cfg.Bucket)
	if cfg.Bucket == "" {
		return nil, errors.New("s3 bucket is required")
	}
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "auto"
	}

	loadOpts := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(region)}
	if strings.TrimSpace(cfg.AccessKeyID) != "" || strings.TrimSpace(cfg.SecretAccessKey) != "" {
		loadOpts = append(loadOpts, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.ForcePathStyle
		if endpoint := strings.TrimSpace(cfg.Endpoint); endpoint != "" {
			o.EndpointResolver = s3.EndpointResolverFromURL(endpoint)
		}
	})

	return &S3Store{client: client, bucket: cfg.Bucket}, nil
}

func (s *S3Store) Put(ctx context.Context, key, contentType string, body []byte) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("blob key is required")
	}
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(contentType),
	})
	return err
}

func (s *S3Store) Get(ctx context.Context, key string) ([]byte, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("blob key is required")
	}
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

func (s *S3Store) Delete(ctx context.Context, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}
