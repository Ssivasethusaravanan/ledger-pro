package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// ---------------------------------------------------------------------------
// Storage Configuration
// ---------------------------------------------------------------------------

// StorageConfig holds credentials for Cloudflare R2 (S3-compatible).
type StorageConfig struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Endpoint        string
}

// ---------------------------------------------------------------------------
// StorageService
// ---------------------------------------------------------------------------

// StorageService encapsulates AWS SDK v2 interactions with Cloudflare R2.
type StorageService struct {
	client         *s3.Client
	presignClient  *s3.PresignClient
	bucketName     string
	logger         *slog.Logger
	urlExpiry      time.Duration
}

// NewStorageService constructs a StorageService configured for Cloudflare R2.
func NewStorageService(ctx context.Context, cfg StorageConfig, logger *slog.Logger) (*StorageService, error) {
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" || cfg.Endpoint == "" {
		return nil, fmt.Errorf("storage configuration is incomplete")
	}

	// Custom endpoint resolver for Cloudflare R2
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: cfg.Endpoint,
		}, nil
	})

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
		config.WithRegion("auto"), // R2 uses 'auto' region
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)
	presignClient := s3.NewPresignClient(client)

	return &StorageService{
		client:        client,
		presignClient: presignClient,
		bucketName:    cfg.BucketName,
		logger:        logger.With(slog.String("component", "storage_service")),
		urlExpiry:     15 * time.Minute, // Presigned URLs valid for 15 mins
	}, nil
}

// UploadFile streams an io.Reader to R2 and returns the unique object key.
func (s *StorageService) UploadFile(ctx context.Context, file io.Reader, objectKey string, contentType string) error {
	start := time.Now()

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(objectKey),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to upload file to R2",
			slog.String("object_key", objectKey),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("put object: %w", err)
	}

	s.logger.InfoContext(ctx, "file uploaded to R2",
		slog.String("object_key", objectKey),
		slog.Duration("duration", time.Since(start)),
	)
	return nil
}

// GetPresignedDownloadURL generates a secure, short-lived URL for downloading a file directly from R2.
func (s *StorageService) GetPresignedDownloadURL(ctx context.Context, objectKey string) (string, error) {
	req, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(objectKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = s.urlExpiry
	})
	if err != nil {
		return "", fmt.Errorf("presign object: %w", err)
	}

	return req.URL, nil
}
