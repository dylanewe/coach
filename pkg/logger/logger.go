package logger

import (
	"go.uber.org/zap"
)

// New creates a production Zap logger.
func New() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.DisableStacktrace = true
	return cfg.Build()
}
