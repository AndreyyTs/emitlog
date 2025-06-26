# EmitLog - Conditional Log Buffering for Go

EmitLog is a Go package that implements conditional log buffering, inspired by Microsoft .NET's log buffering functionality. It allows buffering logs in memory and outputting them only when certain conditions are met, reducing log noise while ensuring all important events are captured.

The package uses [zerolog](https://github.com/rs/zerolog) as the foundation for structured logging, providing high performance and flexibility in log formatting.

## Features

- **Conditional Log Output**: Buffer logs and output them only on errors, warnings, or with configurable probability
- **Memory Efficient**: Configurable buffer size with automatic overflow handling
- **HTTP Middleware**: Ready-to-use middleware for web applications
- **Context Integration**: Seamless integration with Go's context package
- **Built on Zerolog**: Uses the powerful and performant [zerolog](https://github.com/rs/zerolog) library as the foundation for structured logging

## Installation

```bash
go get github.com/rs/zerolog
go get github.com/emitlog/emitlog
```

EmitLog requires the `zerolog` library to function. Make sure you have a compatible version installed.

## Quick Start

### Basic HTTP Middleware Usage

```go
func main() {
    // Setup zerolog
    zerolog.TimeFieldFormat = time.RFC3339
    globalWriter := zerolog.ConsoleWriter{
        Out:        os.Stderr,
        TimeFormat: "15:04:05",
    }
    log.Logger = log.Output(globalWriter)

    // Set logging level
    zerolog.SetGlobalLevel(zerolog.DebugLevel)

    // Create HTTP server
    mux := http.NewServeMux()

    // Register handlers
    mux.HandleFunc("/calculate", CalculateHandler)
    mux.HandleFunc("/health", HealthHandler)
    mux.HandleFunc("/error", ErrorHandler)

    // Create emitlog configuration
    config := emitlog.DefaultConfig()
    config.FlushOnWarn = false // Skip Warn level
    config.SaveRate = 10.0     // Save 10% of successful requests

    // Apply logging middleware
    handler := emitlog.Middleware(config, globalWriter)(mux)

    // Start server
    port := ":8080"
    log.Info().
        Str("port", port).
        Float64("save_rate", config.SaveRate).
        Bool("buffering_enabled", config.BufferingEnabled).
        Bool("flush_on_error", config.FlushOnError).
        Msg("Starting HTTP server")

    if err := http.ListenAndServe(port, handler); err != nil {
        log.Fatal().Err(err).Msg("Server failed to start")
    }
}
```

### Custom Configuration

```go
config := emitlog.Config{
    SaveRate:         10.0,      // Save 10% of successful requests
    BufferingEnabled: true,      // Enable buffering
    FlushOnError:     true,      // Always flush on errors
    FlushOnWarn:      true,      // Also flush on warnings
    BufferSize:       64 * 1024, // 64KB buffer
}

middleware := emitlog.Middleware(config, os.Stdout)
```

## Configuration Parameters

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `SaveRate` | `float64` | `5.0` | Percentage (0-100) of successful requests to save logs for |
| `BufferingEnabled` | `bool` | `true` | Whether to enable log buffering |
| `FlushOnError` | `bool` | `true` | Immediately flush buffer on error-level logs |
| `FlushOnWarn` | `bool` | `false` | Immediately flush buffer on warning-level logs |
| `BufferSize` | `int` | `65536` | Maximum buffer size in bytes (64KB) |

## How It Works

### Buffering Strategy

EmitLog analyzes the JSON structure of zerolog logs to determine the logging level:

1. **Normal Operation**: Zerolog logs are buffered in memory instead of immediate writing
2. **Error Conditions**: When an error or warning occurs (depending on configuration), all buffered logs are immediately flushed
3. **Successful Completion**: Logs are saved based on the probability specified in `SaveRate`
4. **Buffer Overflow**: If buffer size exceeds `BufferSize`, logs are automatically flushed

The package works at the writer level, intercepting zerolog JSON structures before they are written to the final writer.

### Log Levels and Behavior

EmitLog recognizes standard zerolog logging levels from the JSON structure:

- **Error/Fatal/Panic**: Always trigger immediate flush if `FlushOnError` is true
- **Warning**: Triggers immediate flush if `FlushOnWarn` is true
- **Info/Debug/Trace**: Buffered normally

Level parsing occurs by analyzing the `"level"` field in the zerolog JSON structure.

### Request Lifecycle

```
Request Start → Buffer Logs → [Error/Warning?] → [Yes: Flush All] → [No: Continue]
                                 ↓
Request End → [Success?] → [Yes: Save by Probability] → [No: Always Save]
```

## Advanced Usage

### Custom BufferingWriter

```go
import (
    "bytes"
    "os"
    "github.com/yourusername/emitlog"
)

func customLogging() {
    config := emitlog.DefaultConfig()
    requestID := "custom-request-123"
    
    writer := emitlog.NewBufferingWriter(os.Stdout, config, requestID)
    
    // Write logs
    writer.Write([]byte(`{"level":"info","msg":"buffered message"}`))
    writer.Write([]byte(`{"level":"debug","msg":"another buffered message"}`))
    
    // Simulate successful completion
    writer.Finalize(true) // Saved only if random probability matches
    
    // Or force flush
    writer.Flush()
}
```

### Context-Aware Logging

```go
import (
    "context"
    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
    "github.com/yourusername/emitlog"
)

func businessLogic(ctx context.Context) {
    // Get configured zerolog logger from context
    logger := emitlog.GetLoggerFromContext(ctx)
    
    // Use standard zerolog API for structured logging
    logger.Info().
        Str("operation", "user_creation").
        Str("user_type", "premium").
        Msg("Starting user creation")
    
    // Detailed logging with additional fields
    logger.Debug().
        Str("step", "validation").
        Int("attempt", 1).
        Msg("Validating input data")
    
    // Your business logic here
    if err := createUser(); err != nil {
        // This error will trigger immediate flush of all buffered logs
        logger.Error().
            Err(err).
            Str("operation", "user_creation").
            Msg("Failed to create user")
        return
    }
    
    // Logging with metrics
    logger.Info().
        Str("operation", "user_creation").
        Dur("duration", time.Since(start)).
        Msg("User created successfully")
    // These logs will be saved only based on SaveRate
}
```

## Performance Considerations

- **Memory Usage**: Each request maintains its own buffer (64KB max by default)
- **CPU Overhead**: Minimal parsing overhead for determining log levels
- **I/O Reduction**: Significantly reduces write operations for successful requests
- **Concurrency Safety**: All operations are thread-safe with mutex protection

## Best Practices

1. **Set Appropriate SaveRate**: Start with 5-10% for production systems
2. **Monitor Buffer Size**: Adjust based on typical log volume per request
3. **Use FlushOnWarn Carefully**: Can significantly increase log volume
4. **Request ID Tracking**: Each request gets a unique ID for correlation
5. **Error Context**: When errors occur, you get full context from buffered logs

## Monitoring and Debugging

The package automatically adds request metadata to logs using zerolog's structured logging capabilities:

```json
{
  "level": "info",
  "time": "2025-06-26T14:30:45Z",
  "request_id": "123e4567-e89b-12d3-a456-426614174000",
  "method": "GET",
  "path": "/api/users",
  "remote_addr": "192.168.1.1:54321",
  "msg": "Request started"
}
```

All fields are added using zerolog.Context, ensuring consistent log structure.

## Comparison with Traditional Logging

| Aspect | Traditional Logging | EmitLog Buffering |
|--------|-------------------|------------------|
| Log Volume | High (all requests) | Low (selective output) |
| Error Context | Limited to error time | Full request context |
| Performance | Constant I/O overhead | Reduced I/O for successful requests |
| Storage Cost | High | Significantly reduced |
| Debugging | Requires log aggregation | Built-in request correlation |

## Usage Examples

### Web API with Buffering

```go
func apiHandler(w http.ResponseWriter, r *http.Request) {
    logger := emitlog.GetLoggerFromContext(r.Context())
    
    logger.Debug().Msg("Starting API request processing")
    
    // Input validation
    if r.Method != http.MethodPost {
        logger.Warn().Str("method", r.Method).Msg("Unsupported HTTP method")
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return // Logs will be flushed due to 405 status
    }
    
    // Data processing
    logger.Info().Msg("Validation passed successfully")
    
    // If everything succeeds, logs are saved only with SaveRate probability
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status": "success"}`))
}
```

### Microservice with Detailed Logging

```go
func processOrder(ctx context.Context, orderID string) error {
    logger := emitlog.GetLoggerFromContext(ctx)
    
    logger.Info().Str("order_id", orderID).Msg("Starting order processing")
    
    // Detailed logging of each step
    logger.Debug().Msg("Checking inventory availability")
    if err := checkInventory(orderID); err != nil {
        logger.Error().Err(err).Msg("Insufficient inventory")
        return err // All previous logs will be flushed
    }
    
    logger.Debug().Msg("Processing payment")
    if err := processPayment(orderID); err != nil {
        logger.Error().Err(err).Msg("Payment processing failed")
        return err // All previous logs will be flushed
    }
    
    logger.Debug().Msg("Creating database record")
    if err := createOrderRecord(orderID); err != nil {
        logger.Error().Err(err).Msg("Failed to create order record")
        return err
    }
    
    logger.Info().Str("order_id", orderID).Msg("Order processed successfully")
    // On success, all these debug logs are saved only with SaveRate probability
    return nil
}
```

## Environment-Specific Configuration

### Development
```go
devConfig := emitlog.Config{
    SaveRate:         100.0, // Save all logs
    BufferingEnabled: false, // Disable buffering for immediate output
    FlushOnError:     true,
    FlushOnWarn:      true,
    BufferSize:       64 * 1024,
}
```

### Testing
```go
testConfig := emitlog.Config{
    SaveRate:         50.0, // Save half of successful requests
    BufferingEnabled: true,
    FlushOnError:     true,
    FlushOnWarn:      true,
    BufferSize:       32 * 1024,
}
```

### Production
```go
prodConfig := emitlog.Config{
    SaveRate:         5.0,  // Save only 5% of successful requests
    BufferingEnabled: true,
    FlushOnError:     true,
    FlushOnWarn:      false, // Don't flush on warnings in production
    BufferSize:       128 * 1024,
}
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.


# EmitLog - Условная Буферизация Логов для Go

EmitLog — это Go пакет, который реализует условную буферизацию логов, вдохновленный функциональностью буферизации логов Microsoft .NET. Он позволяет буферизовать логи в памяти и выводить их только при выполнении определенных условий, уменьшая шум в логах и обеспечивая при этом захват всех важных событий.

Пакет использует [zerolog](https://github.com/rs/zerolog) как основу для структурированного логирования, что обеспечивает высокую производительность и гибкость в форматировании логов.

## Возможности

- **Условный Вывод Логов**: Буферизация логов и их вывод только при ошибках, предупреждениях или с настраиваемой вероятностью
- **Эффективное Использование Памяти**: Настраиваемый размер буфера с автоматической обработкой переполнения
- **HTTP Middleware**: Готовый к использованию middleware для веб-приложений
- **Работа с Контекстом**: Бесшовная интеграция с пакетом context Go
- **Основан на Zerolog**: Использует мощную и производительную библиотеку [zerolog](https://github.com/rs/zerolog) как основу для структурированного логирования

## Установка

```bash
go get github.com/rs/zerolog
go get github.com/emitlog/emitlog
```

EmitLog требует библиотеку `zerolog` для работы. Убедитесь, что у вас установлена совместимая версия.

## Быстрый Старт

### Базовое Использование HTTP Middleware

```go
func main() {
    // Настройка zerolog
    zerolog.TimeFieldFormat = time.RFC3339
    globalWriter := zerolog.ConsoleWriter{
        Out:        os.Stderr,
        TimeFormat: "15:04:05",
    }
    log.Logger = log.Output(globalWriter)

    // Устанавливаем уровень логирования
    zerolog.SetGlobalLevel(zerolog.DebugLevel)

    // Создаем HTTP сервер
    mux := http.NewServeMux()

    // Регистрируем обработчики
    mux.HandleFunc("/calculate", CalculateHandler)
    mux.HandleFunc("/health", HealthHandler)
    mux.HandleFunc("/error", ErrorHandler)

    // Создаем конфигурацию emitlog
    config := emitlog.DefaultConfig()
    config.FlushOnWarn = false // Пропускаем Warn
    config.SaveRate = 10.0     // Сохранять 10% успешных запросов

    // Применяем middleware для логирования
    handler := emitlog.Middleware(config, globalWriter)(mux)

    // Запускаем сервер
    port := ":8080"
    log.Info().
        Str("port", port).
        Float64("save_rate", config.SaveRate).
        Bool("buffering_enabled", config.BufferingEnabled).
        Bool("flush_on_error", config.FlushOnError).
        Msg("Starting HTTP server")

    if err := http.ListenAndServe(port, handler); err != nil {
        log.Fatal().Err(err).Msg("Server failed to start")
    }
}
```

### Пользовательская Конфигурация

```go
config := emitlog.Config{
    SaveRate:         10.0,      // Сохранять 10% успешных запросов
    BufferingEnabled: true,      // Включить буферизацию
    FlushOnError:     true,      // Всегда сбрасывать при ошибках
    FlushOnWarn:      true,      // Также сбрасывать при предупреждениях
    BufferSize:       64 * 1024, // Буфер 64КБ
}

middleware := emitlog.Middleware(config, os.Stdout)
```

## Параметры Конфигурации

| Поле | Тип | По умолчанию | Описание |
|------|-----|--------------|----------|
| `SaveRate` | `float64` | `5.0` | Процент (0-100) успешных запросов для сохранения логов |
| `BufferingEnabled` | `bool` | `true` | Включить ли буферизацию логов |
| `FlushOnError` | `bool` | `true` | Немедленно сбрасывать буфер при логах уровня error |
| `FlushOnWarn` | `bool` | `false` | Немедленно сбрасывать буфер при логах уровня warning |
| `BufferSize` | `int` | `65536` | Максимальный размер буфера в байтах (64КБ) |

## Как Это Работает

### Стратегия Буферизации

EmitLog анализирует JSON-структуру логов zerolog для определения уровня логирования:

1. **Обычная Работа**: Логи zerolog буферизуются в памяти вместо немедленной записи
2. **Условия Ошибок**: При возникновении ошибки или предупреждения (в зависимости от конфигурации) все буферизованные логи немедленно сбрасываются
3. **Успешное Завершение**: Логи сохраняются на основе вероятности, заданной в `SaveRate`
4. **Переполнение Буфера**: Если размер буфера превышает `BufferSize`, логи автоматически сбрасываются

Пакет работает на уровне вывода (Writer), перехватывая JSON-структуры zerolog перед их записью в финальный writer.

### Уровни Логов и Поведение

EmitLog распознает стандартные уровни логирования zerolog из JSON структуры:

- **Error/Fatal/Panic**: Всегда вызывают немедленный сброс, если `FlushOnError` равно true
- **Warning**: Вызывает немедленный сброс, если `FlushOnWarn` равно true
- **Info/Debug/Trace**: Буферизуются обычным образом

Парсинг уровня происходит путем анализа поля `"level"` в JSON структуре логов zerolog.

### Жизненный Цикл Запроса

```
Начало Запроса → Буферизация Логов → [Ошибка/Предупреждение?] → [Да: Сброс Всех] → [Нет: Продолжение]
                                        ↓
Конец Запроса → [Успех?] → [Да: Сохранение по Вероятности] → [Нет: Всегда Сохранять]
```

## Продвинутое Использование

### Пользовательский BufferingWriter

```go
import (
    "bytes"
    "os"
    "github.com/yourusername/emitlog"
)

func customLogging() {
    config := emitlog.DefaultConfig()
    requestID := "custom-request-123"
    
    writer := emitlog.NewBufferingWriter(os.Stdout, config, requestID)
    
    // Записываем логи
    writer.Write([]byte(`{"level":"info","msg":"буферизованное сообщение"}`))
    writer.Write([]byte(`{"level":"debug","msg":"еще одно буферизованное сообщение"}`))
    
    // Имитируем успешное завершение
    writer.Finalize(true) // Сохраняется только если случайная вероятность совпадает
    
    // Или принудительный сброс
    writer.Flush()
}
```

```go
import (
    "context"
    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
    "github.com/yourusername/emitlog"
)

func businessLogic(ctx context.Context) {
    // Получаем настроенный zerolog logger из контекста
    logger := emitlog.GetLoggerFromContext(ctx)
    
    // Используем стандартный API zerolog для структурированного логирования
    logger.Info().
        Str("operation", "user_creation").
        Str("user_type", "premium").
        Msg("Начинаем создание пользователя")
    
    // Детальное логирование с дополнительными полями
    logger.Debug().
        Str("step", "validation").
        Int("attempt", 1).
        Msg("Проверяем входные данные")
    
    // Ваша бизнес-логика здесь
    if err := createUser(); err != nil {
        // Эта ошибка вызовет немедленный сброс всех буферизованных логов
        logger.Error().
            Err(err).
            Str("operation", "user_creation").
            Msg("Не удалось создать пользователя")
        return
    }
    
    // Логирование с метриками
    logger.Info().
        Str("operation", "user_creation").
        Dur("duration", time.Since(start)).
        Msg("Пользователь успешно создан")
    // Эти логи будут сохранены только на основе SaveRate
}
```

## Соображения Производительности

- **Использование Памяти**: Каждый запрос поддерживает свой собственный буфер (по умолчанию максимум 64КБ)
- **Накладные Расходы CPU**: Минимальные накладные расходы на парсинг для определения уровней логов
- **Снижение I/O**: Значительно сокращает операции записи для успешных запросов
- **Безопасность Параллельности**: Все операции потокобезопасны с защитой mutex

## Лучшие Практики

1. **Установите Подходящий SaveRate**: Начните с 5-10% для продакшн систем
2. **Мониторьте Размер Буфера**: Настройте в зависимости от типичного объема логов на запрос
3. **Осторожно Используйте FlushOnWarn**: Может значительно увеличить объем логов
4. **Отслеживание ID Запроса**: Каждый запрос получает уникальный ID для корреляции
5. **Контекст Ошибок**: При возникновении ошибок вы получаете полный контекст из буферизованных логов

## Мониторинг и Отладка

Пакет автоматически добавляет метаданные запроса к логам используя возможности структурированного логирования zerolog:

```json
{
  "level": "info",
  "time": "2025-06-26T14:30:45Z",
  "request_id": "123e4567-e89b-12d3-a456-426614174000",
  "method": "GET",
  "path": "/api/users",
  "remote_addr": "192.168.1.1:54321",
  "msg": "Запрос начат"
}
```

Все поля добавляются с помощью zerolog.Context, что обеспечивает консистентность структуры логов.

## Сравнение с Традиционным Логированием

| Аспект | Традиционное Логирование | EmitLog Буферизация |
|--------|-------------------------|-------------------|
| Объем Логов | Высокий (все запросы) | Низкий (селективный вывод) |
| Контекст Ошибок | Ограничен временем ошибки | Полный контекст запроса |
| Производительность | Постоянные накладные расходы I/O | Сниженные I/O для успешных запросов |
| Стоимость Хранения | Высокая | Значительно снижена |
| Отладка | Требует агрегации логов | Встроенная корреляция запросов |

## Примеры Использования

### Веб-API с Буферизацией

```go
func apiHandler(w http.ResponseWriter, r *http.Request) {
    logger := emitlog.GetLoggerFromContext(r.Context())
    
    logger.Debug().Msg("Начинаем обработку API запроса")
    
    // Валидация входных данных
    if r.Method != http.MethodPost {
        logger.Warn().Str("method", r.Method).Msg("Неподдерживаемый HTTP метод")
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return // Логи будут сброшены из-за статуса 405
    }
    
    // Обработка данных
    logger.Info().Msg("Валидация прошла успешно")
    
    // Если все прошло успешно, логи сохранятся только с вероятностью SaveRate
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status": "success"}`))
}
```

### Микросервис с Детальным Логированием

```go
func processOrder(ctx context.Context, orderID string) error {
    logger := emitlog.GetLoggerFromContext(ctx)
    
    logger.Info().Str("order_id", orderID).Msg("Начинаем обработку заказа")
    
    // Детальное логирование каждого шага
    logger.Debug().Msg("Проверяем наличие товаров")
    if err := checkInventory(orderID); err != nil {
        logger.Error().Err(err).Msg("Недостаточно товаров на складе")
        return err // Все предыдущие логи будут сброшены
    }
    
    logger.Debug().Msg("Обрабатываем платеж")
    if err := processPayment(orderID); err != nil {
        logger.Error().Err(err).Msg("Ошибка обработки платежа")
        return err // Все предыдущие логи будут сброшены
    }
    
    logger.Debug().Msg("Создаем запись в базе данных")
    if err := createOrderRecord(orderID); err != nil {
        logger.Error().Err(err).Msg("Ошибка создания записи заказа")
        return err
    }
    
    logger.Info().Str("order_id", orderID).Msg("Заказ успешно обработан")
    // При успехе все эти debug логи сохранятся только с вероятностью SaveRate
    return nil
}
```

## Настройка для Разных Сред

### Разработка
```go
devConfig := emitlog.Config{
    SaveRate:         100.0, // Сохранять все логи
    BufferingEnabled: false, // Отключить буферизацию для немедленного вывода
    FlushOnError:     true,
    FlushOnWarn:      true,
    BufferSize:       64 * 1024,
}
```

### Тестирование
```go
testConfig := emitlog.Config{
    SaveRate:         50.0, // Сохранять половину успешных запросов
    BufferingEnabled: true,
    FlushOnError:     true,
    FlushOnWarn:      true,
    BufferSize:       32 * 1024,
}
```

### Продакшн
```go
prodConfig := emitlog.Config{
    SaveRate:         5.0,  // Сохранять только 5% успешных запросов
    BufferingEnabled: true,
    FlushOnError:     true,
    FlushOnWarn:      false, // Не сбрасывать на предупреждения в продакшн
    BufferSize:       128 * 1024,
}
```


## Лицензия

Этот проект лицензирован под лицензией MIT - см. файл LICENSE для деталей.
