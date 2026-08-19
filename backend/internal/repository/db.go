package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kn-system/internal/config"
	"kn-system/pkg/logger"

	"go.uber.org/zap"
)

// NewDB opens a GORM connection to Postgres and applies pending migrations.
// Migration files live in /migrations and are applied in lexical order, so
// schema versioning is append-only and reviewable in the repo.
func NewDB(ctx context.Context, cfg config.DatabaseConfig, migrationsDir string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}
	if err := db.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	if err := applyMigrations(ctx, db, migrationsDir); err != nil {
		return nil, fmt.Errorf("apply migrations: %w", err)
	}
	logger.L.Info("database ready")
	return db, nil
}

// applyMigrations runs every .sql file under dir in order. It is deliberately
// simple (no migration tracking table) so the system is idempotent: files use
// CREATE ... IF NOT EXISTS and are safe to re-run. This is appropriate for a
// greenfield system; a tracking table can be added without changing call sites.
func applyMigrations(ctx context.Context, db *gorm.DB, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir %s: %w", dir, err)
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		if err := db.WithContext(ctx).Exec(string(b)).Error; err != nil {
			return fmt.Errorf("apply %s: %w", f, err)
		}
		logger.L.Info("applied migration", zap.String("file", filepath.Base(f)))
	}
	return nil
}

// NewRedis opens a Redis client. Redis is used as the response cache and as an
// optional queue backend; if it is unreachable, callers that cache should
// degrade gracefully (cache misses are not fatal).
func NewRedis(cfg config.RedisConfig) (*redis.Client, error) {
	c := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	if err := c.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return c, nil
}
