package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"kn-system/pkg/logger"

	"go.uber.org/zap"
)

// LocalStorage persists objects on the local filesystem. It is the default
// provider so the system runs without external object storage dependencies
// during development and in the single-container docker profile.
type LocalStorage struct {
	root string
}

// NewLocal creates a LocalStorage rooted at dir, creating it if missing. The
// directory is created eagerly so the first upload cannot fail on a missing
// root.
func NewLocal(dir string) (*LocalStorage, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}
	return &LocalStorage{root: dir}, nil
}

func (s *LocalStorage) Save(ctx context.Context, key string, r io.Reader, contentType string, size int64) error {
	path := s.pathFor(key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir for %s: %w", key, err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", key, err)
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("write %s: %w", key, err)
	}
	logger.L.Debug("stored object", zap.String("key", key), zap.String("path", path))
	return nil
}

func (s *LocalStorage) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	f, err := os.Open(s.pathFor(key))
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", key, err)
	}
	return f, nil
}

func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	err := os.Remove(s.pathFor(key))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete %s: %w", key, err)
	}
	return nil
}

// pathFor joins the root with the key, preventing escape via a cleaned absolute
// path so callers cannot traverse outside the storage root.
func (s *LocalStorage) pathFor(key string) string {
	clean := filepath.Clean("/" + key) // normalize and root-bound
	return filepath.Join(s.root, clean)
}
