package emitlog

// Config contains logger settings
// Config содержит настройки для логгера
type Config struct {
	SaveRate         float64 // Log save probability on successful execution (0-100) / Вероятность сохранения логов при успешном выполнении (0-100)
	BufferingEnabled bool    // Whether buffering is enabled / Включена ли буферизация
	FlushOnError     bool    // Whether to flush buffer on error / Сбрасывать ли буфер при ошибке
	FlushOnWarn      bool    // Whether to flush buffer on warning / Сбрасывать ли буфер при предупреждении
	BufferSize       int     // Maximum buffer size in bytes / Максимальный размер буфера в байтах
}

// DefaultConfig returns default configuration
// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() Config {
	return Config{
		SaveRate:         5.0,       // Save 05% of requests on successful execution / 5% запросов сохраняем при успешном выполнении
		BufferingEnabled: true,      // Enable buffering / Включить буферизацию
		FlushOnError:     true,      // Flush buffer on error / Сбрасывать буфер при ошибке
		FlushOnWarn:      false,     // Don't flush buffer on warning / Не сбрасывать буфер при предупреждении
		BufferSize:       64 * 1024, // 64 КБ buffer size / Размер буфера 64 КБ
	}
}
