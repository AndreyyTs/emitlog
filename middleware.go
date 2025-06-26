package emitlog

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// responseWriter wraps http.ResponseWriter to track status code
// responseWriter оборачивает http.ResponseWriter для отслеживания статус-кода
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}

// Middleware creates configurable middleware for logging
// Middleware создает настраиваемый middleware для логирования
func Middleware(config Config, globalWriter io.Writer) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := uuid.New().String()
			start := time.Now()

			// Create buffering writer / Создаем буферизованный writer
			var bufferingWriter *BufferingWriter

			var logger = log.With().
				Str("request_id", requestID).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("remote_addr", r.RemoteAddr).
				Logger()

			if config.BufferingEnabled {
				bufferingWriter = NewBufferingWriter(globalWriter, config, requestID)
				logger = logger.Output(bufferingWriter)
			}

			logger.Info().Msg("Request started")

			// Add logger to context / Добавляем logger в контекст
			ctx := context.WithValue(r.Context(), LoggerKey, &logger)
			r = r.WithContext(ctx)

			// Create wrapper for ResponseWriter to track status
			// Создаем wrapper для ResponseWriter для отслеживания статуса
			wrappedWriter := &responseWriter{
				ResponseWriter: w,
				statusCode:     200, // Default 200 / По умолчанию 200
			}

			// Handle panic / Обрабатываем панику
			success := true
			defer func() {
				if err := recover(); err != nil {
					success = false
					logger.Error().
						Interface("panic", err).
						Msg("Request panicked")

					if bufferingWriter != nil {
						bufferingWriter.Flush() // Force flush on panic / Принудительно сбрасываем при панике
					}

					http.Error(wrappedWriter, "Internal Server Error", http.StatusInternalServerError)
					panic(err) // Re-panic for handling up the stack / Re-panic для обработки выше по стеку
				}
			}()

			// Execute request / Выполняем запрос
			next.ServeHTTP(wrappedWriter, r)

			// Determine success by status code / Определяем успешность по статус-коду
			if wrappedWriter.statusCode >= 400 {
				success = false
			}

			duration := time.Since(start)
			logger.Info().
				Int("status_code", wrappedWriter.statusCode).
				Dur("duration", duration).
				Bool("success", success).
				Msg("Request completed")

			// Finalize log processing / Завершаем обработку логов
			if bufferingWriter != nil {
				bufferingWriter.Finalize(success)
			}
		})
	}
}
