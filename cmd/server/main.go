package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"kn-system/internal/config"
	"kn-system/internal/server"
	"kn-system/pkg/logger"

	"go.uber.org/zap"
)

// main is the process entry point. Its only responsibilities are: load config,
// initialize logging, construct the server, wire OS signals for graceful
// shutdown, and block until shutdown completes. All real work lives in the
// server package so this file stays a thin bootstrap.
func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	logger.Init(os.Getenv("APP_ENV"))
	defer logger.Sync()

	srv, err := server.New(cfg)
	if err != nil {
		logger.L.Fatal("server init failed", zap.Error(err))
	}

	// Run the server on a goroutine so the main goroutine can wait on signals.
	errCh := make(chan error, 1)
	go func() {
		if err := srv.Start(context.Background()); err != nil {
			errCh <- err
		}
	}()

	// Block until a termination signal arrives or the server errors out. This
	// is the single shutdown trigger so we don't race two stop paths.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		logger.L.Fatal("server exited", zap.Error(err))
	case sig := <-stop:
		logger.L.Info("received signal, shutting down", zap.Stringer("signal", sig))
	}

	// Give in-flight requests a bounded window to finish; the spec calls for
	// graceful shutdown without interrupting active requests.
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(cfg.Server.ShutdownTimeout)*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.L.Error("shutdown error", zap.Error(err))
		os.Exit(1)
	}
}
