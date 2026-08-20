package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"kn-system/internal/config"
	"kn-system/pkg/logger"

	"go.uber.org/zap"
)

// MinIOStorage implements Storage against an S3-compatible object store
// (MinIO, or AWS S3 / Alibaba OSS with the right endpoint). It satisfies the
// non-functional requirement that file storage be swappable.
type MinIOStorage struct {
	client *minio.Client
	bucket string
}

// NewMinIO constructs a client and ensures the target bucket exists, creating
// it on first run so operators don't have to pre-provision storage.
func NewMinIO(cfg config.StorageConfig) (*MinIOStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("minio bucket check: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("minio make bucket: %w", err)
		}
		logger.L.Info("created bucket", zap.String("bucket", cfg.Bucket))
	}
	return &MinIOStorage{client: client, bucket: cfg.Bucket}, nil
}

func (s *MinIOStorage) Save(ctx context.Context, key string, r io.Reader, contentType string, size int64) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("minio put %s: %w", key, err)
	}
	return nil
}

func (s *MinIOStorage) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("minio get %s: %w", key, err)
	}
	return obj, nil
}

func (s *MinIOStorage) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("minio delete %s: %w", key, err)
	}
	return nil
}

// New picks the storage backend from configuration. Adding a new provider is a
// case here, not a change to callers.
func New(cfg config.StorageConfig) (Storage, error) {
	switch cfg.Provider {
	case "minio":
		return NewMinIO(cfg)
	default:
		return NewLocal(cfg.LocalDir)
	}
}
