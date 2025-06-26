package emitlog

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ContextKey for passing logger through context
// ContextKey для передачи логгера через контекст
type ContextKey string

const LoggerKey ContextKey = "emitlog"

// GetLoggerFromContext extracts logger from context
// GetLoggerFromContext извлекает логгер из контекста
func GetLoggerFromContext(ctx context.Context) *zerolog.Logger {
	if logger, ok := ctx.Value(LoggerKey).(*zerolog.Logger); ok {
		return logger
	}
	// Return global logger as fallback / Возвращаем глобальный логгер как fallback
	return &log.Logger
}
