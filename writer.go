package emitlog

import (
	"bytes"
	"io"
	"math/rand"
	"sync"

	"github.com/rs/zerolog"
)

// BufferingWriter implements buffered log writing
// BufferingWriter реализует буферизованную запись логов
type BufferingWriter struct {
	mu          sync.RWMutex
	finalWriter io.Writer     // Where to ultimately write logs / Куда в итоге записывать логи
	buffer      *bytes.Buffer // Buffer for temporary storage / Буфер для временного хранения
	shouldFlush bool          // Whether buffer should be flushed / Нужно ли сбросить буфер
	config      Config
	requestID   string
	bufferSize  int
}

// NewBufferingWriter creates a new buffered writer
// NewBufferingWriter создает новый буферизованный writer
func NewBufferingWriter(finalWriter io.Writer, config Config, requestID string) *BufferingWriter {
	return &BufferingWriter{
		finalWriter: finalWriter,
		buffer:      &bytes.Buffer{},
		config:      config,
		requestID:   requestID,
	}
}

// Write implements io.Writer interface
// Write реализует интерфейс io.Writer
func (bw *BufferingWriter) Write(p []byte) (n int, err error) {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	// If already decided to flush, write directly to final writer
	// Если уже решили сбрасывать, пишем сразу в финальный writer
	if bw.shouldFlush {
		return bw.finalWriter.Write(p)
	}

	// Check log level / Проверяем уровень логирования
	level := bw.parseLogLevel(p)

	// Decide if buffer should be flushed / Решаем, нужно ли сбросить буфер
	shouldFlushNow := false
	switch level {
	case zerolog.ErrorLevel, zerolog.FatalLevel, zerolog.PanicLevel:
		if bw.config.FlushOnError {
			shouldFlushNow = true
		}
	case zerolog.WarnLevel:
		if bw.config.FlushOnWarn {
			shouldFlushNow = true
		}
	}

	// If buffer needs to be flushed / Если нужно сбросить буфер
	if shouldFlushNow {
		bw.shouldFlush = true
		// First flush accumulated buffer / Сначала сбрасываем весь накопленный буфер
		if bw.buffer.Len() > 0 {
			bw.finalWriter.Write(bw.buffer.Bytes())
			bw.buffer.Reset()
		}
		// Then write current message / Затем записываем текущее сообщение
		return bw.finalWriter.Write(p)
	}

	// Check buffer size / Проверяем размер буфера
	if bw.bufferSize+len(p) > bw.config.BufferSize {
		// Buffer overflow, flush it / Буфер переполнен, сбрасываем его
		bw.shouldFlush = true
		if bw.buffer.Len() > 0 {
			bw.finalWriter.Write(bw.buffer.Bytes())
			bw.buffer.Reset()
		}
		return bw.finalWriter.Write(p)
	}

	// Add to buffer / Добавляем в буфер
	n, err = bw.buffer.Write(p)
	bw.bufferSize += n
	return n, err
}

// parseLogLevel extracts log level from JSON message
// parseLogLevel извлекает уровень логирования из JSON сообщения
func (bw *BufferingWriter) parseLogLevel(data []byte) zerolog.Level {
	if bytes.Contains(data, []byte(`"level":"trace"`)) {
		return zerolog.TraceLevel
	}
	if bytes.Contains(data, []byte(`"level":"debug"`)) {
		return zerolog.DebugLevel
	}
	if bytes.Contains(data, []byte(`"level":"info"`)) {
		return zerolog.InfoLevel
	}
	if bytes.Contains(data, []byte(`"level":"warn"`)) {
		return zerolog.WarnLevel
	}
	if bytes.Contains(data, []byte(`"level":"error"`)) {
		return zerolog.ErrorLevel
	}
	if bytes.Contains(data, []byte(`"level":"fatal"`)) {
		return zerolog.FatalLevel
	}
	if bytes.Contains(data, []byte(`"level":"panic"`)) {
		return zerolog.PanicLevel
	}
	return zerolog.InfoLevel
}

// Flush forcibly flushes the buffer
// Flush принудительно сбрасывает буфер
func (bw *BufferingWriter) Flush() {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	if bw.buffer.Len() > 0 {
		bw.finalWriter.Write(bw.buffer.Bytes())
		bw.buffer.Reset()
	}
	bw.shouldFlush = true
}

// ShouldSaveOnSuccess decides whether to save logs on successful completion
// ShouldSaveOnSuccess решает, сохранять ли логи при успешном завершении
func (bw *BufferingWriter) ShouldSaveOnSuccess() bool {
	if !bw.config.BufferingEnabled {
		return true
	}

	if bw.config.SaveRate <= 0 {
		return false
	}
	if bw.config.SaveRate >= 100 {
		return true
	}

	return rand.Float64()*100 < bw.config.SaveRate
}

// Finalize completes log processing for the request
// Finalize завершает обработку логов для запроса
func (bw *BufferingWriter) Finalize(success bool) {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	// If already flushed during process (due to errors), do nothing
	// Если уже сбрасывали в процессе (из-за ошибок), ничего не делаем
	if bw.shouldFlush {
		return
	}

	// Decide whether to save logs / Решаем, сохранять ли логи
	shouldSave := false
	if !success {
		// Always save on unsuccessful completion / При неуспешном завершении всегда сохраняем
		shouldSave = true
	} else {
		// On successful completion check settings / При успешном завершении проверяем настройки
		shouldSave = bw.ShouldSaveOnSuccess()
	}

	if shouldSave && bw.buffer.Len() > 0 {
		bw.finalWriter.Write(bw.buffer.Bytes())
	}

	// Clear buffer anyway / Очищаем буфер в любом случае
	bw.buffer.Reset()
}
