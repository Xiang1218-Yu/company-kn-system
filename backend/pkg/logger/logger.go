package logger

import (
	"sync"

	"go.uber.org/zap"
)

// L is the package-level logger. It is initialized once on the first call to
// Init and is safe for concurrent use. Components depend on this global rather
// than threading a logger instance everywhere, which keeps call sites short.
var (
	L     *zap.Logger
	once  sync.Once
)

// Init configures the global logger. It is safe to call multiple times; the
// first call wins. In development it writes human-readable console output; in
// production it writes structured JSON for machine consumption.
func Init(env string) {
	once.Do(func() {
		var logger *zap.Logger
		var err error
		if env == "production" {
			logger, err = zap.NewProduction()
		} else {
			logger, err = zap.NewDevelopment()
		}
		if err != nil {
			panic("failed to initialize logger: " + err.Error())
		}
		L = logger
	})
}

// Sync flushes buffered log entries. Defer this at shutdown.
func Sync() {
	if L != nil {
		_ = L.Sync()
	}
}
