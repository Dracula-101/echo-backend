package r2

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3Types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"shared/pkg/logger"
	"shared/pkg/storage"
)

// Provider implements storage.Provider for Cloudflare R2 (S3-compatible)
type Provider struct {
	client     *s3.Client
	bucket     string
	publicURL  string
	cdnBaseURL string
	useCDN     bool
	log        logger.Logger
}

// Config contains configuration for R2 storage
type Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	PublicURL       string
	CDNBaseURL      string
	UseCDN          bool
	ImageQuality    int
}

// New creates a new R2 storage provider
func New(cfg Config, log logger.Logger) (*Provider, error) {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)

	// Create AWS SDK config for R2
	r2Config, err := config.LoadDefaultConfig(context.Background(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
		config.WithRegion("auto"), // R2 uses "auto" region
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load R2 config: %w", err)
	}

	// Create S3 client with R2 endpoint
	client := s3.NewFromConfig(r2Config, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	return &Provider{
		client:     client,
		bucket:     cfg.Bucket,
		publicURL:  cfg.PublicURL,
		cdnBaseURL: cfg.CDNBaseURL,
		useCDN:     cfg.UseCDN,
		log:        log,
	}, nil
}

// Upload uploads a file to R2
func (r *Provider) Upload(ctx context.Context, key string, data io.Reader, contentType string) (string, error) {
	return r.UploadWithOptions(ctx, key, data, storage.UploadOptions{
		ContentType: contentType,
	})
}

// UploadWithOptions uploads a file with custom options
func (r *Provider) UploadWithOptions(ctx context.Context, key string, data io.Reader, opts storage.UploadOptions) (string, error) {
	// Read all data into memory
	fileData, err := io.ReadAll(data)
	if err != nil {
		return "", fmt.Errorf("failed to read file data: %w", err)
	}

	// Build PutObject input
	input := &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(fileData),
		ContentType: aws.String(opts.ContentType),
	}

	if opts.ContentDisposition != "" {
		input.ContentDisposition = aws.String(opts.ContentDisposition)
	}
	if opts.CacheControl != "" {
		input.CacheControl = aws.String(opts.CacheControl)
	}
	if opts.ACL != "" {
		input.ACL = s3Types.ObjectCannedACL(opts.ACL)
	}
	if len(opts.Metadata) > 0 {
		input.Metadata = opts.Metadata
	}

	// Upload to R2
	_, err = r.client.PutObject(ctx, input)
	if err != nil {
		r.log.Error("Failed to upload to R2", logger.String("key", key), logger.Error(err))
		return "", fmt.Errorf("failed to upload to R2: %w", err)
	}

	// Build URL
	url := r.buildURL(key)
	r.log.Info("File uploaded to R2",
		logger.String("key", key),
		logger.String("url", url),
	)

	return url, nil
}

// Download downloads a file from R2
func (r *Provider) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	result, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		r.log.Error("Failed to download from R2", logger.String("key", key), logger.Error(err))
		return nil, fmt.Errorf("failed to download from R2: %w", err)
	}

	return result.Body, nil
}

// Delete deletes a file from R2
func (r *Provider) Delete(ctx context.Context, key string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		r.log.Error("Failed to delete from R2", logger.String("key", key), logger.Error(err))
		return fmt.Errorf("failed to delete from R2: %w", err)
	}

	r.log.Info("File deleted from R2", logger.String("key", key))
	return nil
}

// DeleteMultiple deletes multiple files from R2
func (r *Provider) DeleteMultiple(ctx context.Context, keys []string) error {
	objects := make([]s3Types.ObjectIdentifier, len(keys))
	for i, key := range keys {
		objects[i] = s3Types.ObjectIdentifier{
			Key: aws.String(key),
		}
	}

	_, err := r.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(r.bucket),
		Delete: &s3Types.Delete{
			Objects: objects,
		},
	})

	if err != nil {
		r.log.Error("Failed to delete multiple objects from R2", logger.Error(err))
		return fmt.Errorf("failed to delete multiple objects: %w", err)
	}

	return nil
}

// Exists checks if a file exists in R2
func (r *Provider) Exists(ctx context.Context, key string) (bool, error) {
	_, err := r.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		// Check if it's a "not found" error
		return false, nil
	}

	return true, nil
}

// GetMetadata retrieves metadata for a file
func (r *Provider) GetMetadata(ctx context.Context, key string) (*storage.Metadata, error) {
	result, err := r.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	return &storage.Metadata{
		Key:          key,
		Size:         *result.ContentLength,
		ContentType:  aws.ToString(result.ContentType),
		LastModified: aws.ToTime(result.LastModified),
		ETag:         aws.ToString(result.ETag),
		Metadata:     result.Metadata,
	}, nil
}

// GeneratePresignedURL generates a presigned URL for temporary access
func (r *Provider) GeneratePresignedURL(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(r.client)

	presignedURL, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiresIn
	})

	if err != nil {
		r.log.Error("Failed to generate presigned URL", logger.String("key", key), logger.Error(err))
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedURL.URL, nil
}

// GenerateUploadURL generates a presigned URL for uploading
func (r *Provider) GenerateUploadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(r.client)

	presignedURL, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiresIn
	})

	if err != nil {
		r.log.Error("Failed to generate upload URL", logger.String("key", key), logger.Error(err))
		return "", fmt.Errorf("failed to generate upload URL: %w", err)
	}

	return presignedURL.URL, nil
}

// ListObjects lists objects with a given prefix
func (r *Provider) ListObjects(ctx context.Context, prefix string, maxKeys int) ([]storage.ObjectInfo, error) {
	maxKeys32 := int32(maxKeys)
	result, err := r.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(r.bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: &maxKeys32,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	objects := make([]storage.ObjectInfo, len(result.Contents))
	for i, obj := range result.Contents {
		objects[i] = storage.ObjectInfo{
			Key:          aws.ToString(obj.Key),
			Size:         *obj.Size,
			LastModified: aws.ToTime(obj.LastModified),
			ETag:         aws.ToString(obj.ETag),
			StorageClass: string(obj.StorageClass),
		}
	}

	return objects, nil
}

// buildURL builds the public URL for a file
func (r *Provider) buildURL(key string) string {
	if r.useCDN && r.cdnBaseURL != "" {
		return fmt.Sprintf("%s/%s", r.cdnBaseURL, key)
	}
	if r.publicURL != "" {
		return fmt.Sprintf("%s/%s", r.publicURL, key)
	}
	return fmt.Sprintf("https://%s.r2.cloudflarestorage.com/%s", r.bucket, key)
}
