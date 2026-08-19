package storage

import (
	"context"
	"io"
)

// Storage abstracts the file backend behind a narrow interface so the service
// layer never knows whether bytes land on local disk, MinIO, or S3. Swapping
// providers (a non-functional requirement) means implementing this interface,
// not editing call sites.
type Storage interface {
	// Save stores the stream under key and returns the object key for later
	// retrieval. The caller owns the returned key and persists it on the
	// document record.
	Save(ctx context.Context, key string, r io.Reader, contentType string, size int64) error
	// Open returns a reader for the stored object. Callers must close it.
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes the object. A missing object is not an error.
	Delete(ctx context.Context, key string) error
}
