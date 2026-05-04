package log

import (
	"context"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	globalLogger *zap.Logger
	once         sync.Once
)

// InitLogger initializes the global zap logger.
// Production by default; set DEBUG=true for human-readable output.
func InitLogger() {
	once.Do(func() {
		var cfg zap.Config
		if os.Getenv("DEBUG") == "true" {
			cfg = zap.NewDevelopmentConfig()
			cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		} else {
			cfg = zap.NewProductionConfig()
			cfg.EncoderConfig.TimeKey = "ts"
			cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		}
		cfg.OutputPaths = []string{"stderr"}
		cfg.ErrorOutputPaths = []string{"stderr"}

		logger, err := cfg.Build()
		if err != nil {
			panic("failed to initialize logger: " + err.Error())
		}
		globalLogger = logger
	})
}

// GetLogger returns the global zap logger.
// Accepts context for future per-request loggers, currently ignored.
func GetLogger(ctx context.Context) *zap.Logger {
	if globalLogger == nil {
		InitLogger()
	}
	return globalLogger
}

// Sync flushes any buffered log entries.
func Sync() {
	if globalLogger != nil {
		_ = globalLogger.Sync()
	}
}
