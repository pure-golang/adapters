# Тесты без t.Parallel() — причины и обоснование

## Общий принцип

`t.Parallel()` нельзя использовать в тестах, которые изменяют глобальное состояние процесса без должной изоляции. Параллельный запуск таких тестов приводит либо к панике в рантайме, либо к гонкам данных и нестабильным результатам.

---

## env/env_test.go

### Причина: `t.Setenv()` несовместим с `t.Parallel()`

Начиная с Go 1.17, вызов `t.Setenv()` в параллельном тесте вызывает **панику**:

```
panic: testing: test using t.Setenv or t.Chdir can not use t.Parallel
```

`t.Setenv()` изменяет переменные окружения глобально для всего процесса. Go запрещает это в параллельных тестах, чтобы предотвратить гонки данных между тестами, читающими те же переменные.

**Тесты без `t.Parallel()` из-за `t.Setenv()`:**

| Тест | Устанавливаемые переменные |
|------|---------------------------|
| `TestInitConfig_ValidStruct` | `HOST`, `PORT`, `REQUIRED` |
| `TestInitConfig_PartialEnvVarsWithDefaults` | `HOST`, `REQUIRED` |
| `TestInitConfig_BoolDefault` | `HOST`, `PORT`, `API_KEY` |
| `TestInitConfig_BoolSet` | `HOST`, `PORT`, `API_KEY`, `ENABLED` |
| `TestInitConfig_EmptyStringField` | `HOST`, `PORT`, `REQUIRED` |
| `TestInitConfig_InvalidPortFormat` | `HOST`, `PORT`, `REQUIRED` |

### Причина: загрязнение окружения через `.env`-файл

Тесты `TestInitConfig_WithDotEnvFile` и `TestInitConfig_EnvVarsOverrideDotEnv` используют `os.Chdir()` для смены рабочей директории и загружают `.env`-файл через `InitConfig`. Библиотека `godotenv` при загрузке файла вызывает `os.Setenv()` — переменные окружения остаются установленными после завершения теста (рабочая директория восстанавливается через `defer`, но переменные окружения — нет).

Это загрязняет окружение для последующих тестов, которые рассчитывают на чистое состояние:

**Тесты без `t.Parallel()` из-за загрязнения окружения:**

| Тест | Ожидание | Что может прийти из .env |
|------|----------|--------------------------|
| `TestInitConfig_MissingRequiredEnvVar` | `HOST` и `PORT` не установлены → ошибка | `HOST=fromdotenv`, `PORT=7000` → тест неожиданно проходит |
| `TestInitConfig_DefaultValues` | `HOST=localhost` (по умолчанию) | `HOST=fromdotenv` → значение не совпадает |

---

## kv/kv_test.go

### Причина: `t.Setenv()` несовместим с `t.Parallel()`

Та же причина, что и в `env/env_test.go`. Все тесты, устанавливающие `KV_PROVIDER`, `REDIS_ADDR`, `REDIS_DB` и другие переменные через `t.Setenv()`, не могут использовать `t.Parallel()`.

**Тесты без `t.Parallel()` из-за `t.Setenv()`:**

| Тест | Устанавливаемые переменные |
|------|---------------------------|
| `TestNewDefault_NoopProvider` | `KV_PROVIDER` |
| `TestNewDefault_RedisProvider_ConnectionError` | `KV_PROVIDER`, `REDIS_ADDR` |
| `TestNewDefault_UnknownProvider` | `KV_PROVIDER` |
| `TestInitDefault_AliasForNewDefault` | `KV_PROVIDER` |
| `TestInitDefault_UnknownProvider` | `KV_PROVIDER` |
| `TestStore_InterfaceImplementation` | `KV_PROVIDER` |
| `TestConfig_WithRedisSettings` | `KV_PROVIDER`, `REDIS_ADDR`, `REDIS_DB`, `REDIS_MAX_RETRIES`, `REDIS_POOL_SIZE` |
| `TestNewDefault_EnvInitError` | `KV_PROVIDER`, `REDIS_DB` |

### Причина: вспомогательная функция `unsetKey()` изменяет глобальное окружение

Функция `unsetKey(t, key)` напрямую вызывает `os.Unsetenv()` и восстанавливает значение через `t.Cleanup()`. Она не проходит через `t.Setenv()`, поэтому Go не вызывает панику — но изменение глобального окружения всё равно создаёт гонки данных при параллельном выполнении.

**Тесты без `t.Parallel()` из-за `unsetKey()`:**

| Тест | Сбрасываемые переменные |
|------|------------------------|
| `TestNewDefault_DefaultProvider` | `KV_PROVIDER` |
| `TestConfig_DefaultValues` | `KV_PROVIDER`, `REDIS_ADDR`, `REDIS_DB`, `REDIS_MAX_RETRIES`, `REDIS_DIAL_TIMEOUT`, `REDIS_READ_TIMEOUT`, `REDIS_WRITE_TIMEOUT`, `REDIS_POOL_SIZE` |

---

## db/pg/pgx/logger_test.go

### Причина: общие переменные между родительским и под-тестами

В родительских тестах объявлены переменные (`records`, `testHandler`, `testLogger`, `ctx`, `pgxLogger`), которые используются в под-тестах. Когда родительский тест имеет `t.Parallel()` и под-тесты также имеют `t.Parallel()`, они могут модифицировать эти переменные параллельно, что создаёт гонки данных.

Пример из `TestLogger_Log`:
```go
func TestLogger_Log(t *testing.T) {
    // Общие переменные для всех под-тестов
    var records []slog.Record
    testHandler := &testHandler{records: &records}
    testLogger := slog.New(testHandler)
    ctx := logger.NewContext(context.Background(), testLogger)
    pgxLogger := &Logger{}

    t.Run("Log with Trace level", func(t *testing.T) {
        t.Parallel()
        records = nil // Гонка данных: несколько под-тестов модифицируют records
        // ...
    })
}
```

**Тесты без `t.Parallel()` из-за общих переменных:**

| Тест | Общие переменные |
|------|------------------|
| `TestLogger_Log` | `records`, `testHandler`, `testLogger`, `ctx`, `pgxLogger` |
| `TestLogger_LogWithDuration` | `records`, `testHandler`, `testLogger`, `ctx`, `pgxLogger` |
| `TestLogger_LogWithNilData` | `records`, `testHandler`, `testLogger`, `ctx`, `pgxLogger` |
| `TestLogger_LogWithComplexData` | `records`, `testHandler`, `testLogger`, `ctx`, `pgxLogger` |
| `TestLogger_LogWithSpecialDurationValues` | `records`, `testHandler`, `testLogger`, `ctx`, `pgxLogger` |
| `TestLogger_LogWithNonDurationTimeValue` | `records`, `testHandler`, `testLogger`, `ctx`, `pgxLogger` |

---

## db/pg/pgx/errors_test.go

### Причина: родительский тест с под-тестами

В `TestErrorIs_AllErrorCodeCombinations` родительский тест имеет `t.Parallel()`, но под-тесты не имеют. Это создаёт несогласованность в выполнении тестов и может привести к гонкам данных при доступе к общим данным.

**Тесты без `t.Parallel()` из-за родительского теста с под-тестами:**

| Тест | Описание |
|------|----------|
| `TestErrorIs_AllErrorCodeCombinations` | Родительский тест с циклом под-тестов |

---

## db/pg/pgx/pg_test.go

### Причина: родительские тесты с под-тестами

В нескольких тестах родительские тесты имеют `t.Parallel()` и содержат под-тесты, которые также могут выполняться параллельно. Это создаёт потенциальные гонки данных и нестабильное поведение тестов.

**Тесты без `t.Parallel()` из-за родительских тестов с под-тестами:**

| Тест | Описание |
|------|----------|
| `TestNewDefault_TraceLogLevelParsing` | Родительский тест с циклом под-тестов |
| `TestNew_InvalidConfig` | Родительский тест с циклом под-тестов |
| `TestDB_Close` | Родительский тест с под-тестами |
| `TestDB_Close_NilPool` | Родительский тест с под-тестами |
| `TestDB_ConnectionPoolOptions` | Родительский тест с циклом под-тестов |
| `TestConfig_DurationValues` | Родительский тест с циклом под-тестов |
| `TestNew_ConnectionFailures` | Родительский тест с циклом под-тестов |
| `TestNewDefault_VariousConfigs` | Родительский тест с циклом под-тестов |
| `TestNewDefault_TracerSetup` | Родительский тест с под-тестами |
| `TestNewDefault_ConnectionFailures` | Родительский тест с циклом под-тестов |
| `TestNewDefault_ConfigVariations` | Родительский тест с под-тестами |

---

## db/pg/pgx/tracing_test.go

### Причина: родительские тесты с под-тестами

Аналогичная проблема с родительскими тестами, которые имеют `t.Parallel()` и содержат под-тесты.

**Тесты без `t.Parallel()` из-за родительских тестов с под-тестами:**

| Тест | Описание |
|------|----------|
| `TestParseTraceLogLevel_ValidLevels` | Родительский тест с циклом под-тестов |
| `TestParseTraceLogLevel_InvalidLevel` | Родительский тест с циклом под-тестов |
| `TestParseTraceLogLevel_ProductionDefaults` | Родительский тест с циклом под-тестов |
| `TestParseTraceLogLevel_DevelopmentDefaults` | Родительский тест с циклом под-тестов |

---

## grpc/middleware/monitoring_test.go

### Причина: глобальное состояние OpenTelemetry

Все тесты в этом файле вызывают `SetupMonitoring()`, который при включенном трейсинге устанавливает глобальное состояние через `otel.SetTextMapPropagator()`. При параллельном выполнении тестов это вызывает race conditions.

**Тесты без `t.Parallel()` из-за глобального состояния OpenTelemetry:**

| Тест | Описание |
|------|----------|
| `TestDefaultMonitoringOptions` | Тестирует настройки по умолчанию |
| `TestSetupMonitoring_AllOptionsEnabled` | Тестирует настройку мониторинга со всеми опциями |
| `TestSetupMonitoring_TracingDisabled` | Тестирует настройку с отключенным трейсингом |
| `TestSetupMonitoring_MetricsDisabled` | Тестирует настройку с отключенными метриками |
| `TestSetupMonitoring_LoggingDisabled` | Тестирует настройку с отключенным логированием |
| `TestSetupMonitoring_AllDisabled` | Тестирует настройку со всеми отключенными опциями |
| `TestSetupMonitoring_StatsHandlerDisabled` | Тестирует настройку без StatsHandler |
| `TestSetupMonitoring_OnlyTracing` | Тестирует настройку только с трейсингом |
| `TestSetupMonitoring_OnlyMetrics` | Тестирует настройку только с метриками |
| `TestSetupMonitoring_OnlyLogging` | Тестирует настройку только с логированием |
| `TestSetupMonitoring_InterceptorsAreFunctional` | Тестирует функциональность интерцепторов |
| `TestSetupMonitoring_NilLogger` | Тестирует настройку с nil logger |
| `TestSetupMonitoring_TracingSetsPropagator` | Тестирует установку пропагатора трейсинга |
| `TestSetupMonitoring_Context` | Тестирует передачу контекста |
| `TestMonitoringOptions_StructFields` | Тестирует поля структуры MonitoringOptions |
| `TestSetupMonitoring_InterceptorsOrder` | Тестирует порядок интерцепторов |
| `TestSetupMonitoring_StreamInterceptorsOrder` | Тестирует порядок stream интерцепторов |

---

## grpc/middleware/tracing_test.go

### Причина: глобальное состояние OpenTelemetry

Некоторые тесты в этом файле вызывают `otel.SetTextMapPropagator()` и `otel.SetTracerProvider()`, которые устанавливают глобальное состояние OpenTelemetry. При параллельном выполнении тестов это вызывает race conditions.

**Тесты без `t.Parallel()` из-за глобального состояния OpenTelemetry:**

| Тест | Описание |
|------|----------|
| `TestTracingUnaryInterceptor_CreatesSpan` | Тестирует создание спана |
| `TestTracingUnaryInterceptor_WithError` | Тестирует обработку ошибок |
| `TestTracingUnaryInterceptor_ExtractsTraceContext` | Тестирует извлечение контекста трейсинга |
| `TestTracingUnaryInterceptor_NoMetadata` | Тестирует обработку без метаданных |
| `TestTracingStreamInterceptor_CreatesSpan` | Тестирует создание спана для stream |
| `TestTracingStreamInterceptor_WithError` | Тестирует обработку ошибок для stream |
| `TestTracingStreamInterceptor_DetectsStreamType` | Тестирует определение типа stream |
| `TestTracingStreamInterceptor_ExtractsTraceContext` | Тестирует извлечение контекста трейсинга для stream |
| `TestTracingUnaryInterceptor_SpanAttributes` | Тестирует атрибуты спана |
| `TestTracingStreamInterceptor_SpanAttributes` | Тестирует атрибуты спана для stream |
| `TestTracingUnaryInterceptor_DifferentStatusCodes` | Тестирует разные статус коды |
| `TestMetadataTextMapPropagator_Propagation` | Тестирует пропагацию контекста |

**Тесты с `t.Parallel()` (не используют глобальное состояние):**

| Тест | Описание |
|------|----------|
| `TestSplitMethodName_ValidPath` | Тестирует разбор имени метода (валидный путь) |
| `TestSplitMethodName_InvalidPath` | Тестирует разбор имени метода (невалидный путь) |
| `TestMetadataTextMapPropagator` | Тестирует создание пропагатора |
| `TestMetadataSupplier_Get` | Тестирует метод Get |
| `TestMetadataSupplier_Set` | Тестирует метод Set |
| `TestMetadataSupplier_Keys` | Тестирует метод Keys |
| `TestMetadataSupplier_Empty` | Тестирует работу с пустыми метаданными |
| `TestWrappedServerStream_Context` | Тестирует метод Context |
| `TestWrappedServerStream_ContextWithValue` | Тестирует контекст со значением |
| `TestSplitMethodName_URLSafe` | Тестирует разбор URL-safe путей |

---

## grpc/middleware/metrics_test.go

### Причина: глобальное состояние OpenTelemetry

Большинство тестов в этом файле вызывают `otel.SetMeterProvider()` и `otel.SetTracerProvider()`, которые устанавливают глобальное состояние OpenTelemetry. При параллельном выполнении тестов это вызывает race conditions.

**Тесты без `t.Parallel()` из-за глобального состояния OpenTelemetry:**

| Тест | Описание |
|------|----------|
| `TestMetricsUnaryInterceptor_Success` | Тестирует метрики для успешных запросов |
| `TestMetricsUnaryInterceptor_WithError` | Тестирует метрики для запросов с ошибками |
| `TestMetricsUnaryInterceptor_WithProtoMessage` | Тестирует метрики с protobuf сообщениями |
| `TestMetricsUnaryInterceptor_WithNonProtoMessage` | Тестирует метрики с non-protobuf сообщениями |
| `TestMetricsUnaryInterceptor_DifferentStatusCodes` | Тестирует разные статус коды |
| `TestMetricsUnaryInterceptor_Context` | Тестирует передачу контекста |
| `TestMetricsStreamInterceptor_Success` | Тестирует метрики для успешных stream |
| `TestMetricsStreamInterceptor_WithError` | Тестирует метрики для stream с ошибками |
| `TestMetricsStreamInterceptor_StreamTypes` | Тестирует разные типы stream |
| `TestMetricsStreamInterceptor_Context` | Тестирует передачу контекста для stream |
| `TestMetricsUnaryInterceptor_DifferentMessageSizes` | Тестирует разные размеры сообщений |
| `TestMetricsUnaryInterceptor_WithPanic` | Тестирует поведение при панике |
| `TestMetricsStreamInterceptor_WithPanic` | Тестирует поведение при панике для stream |
| `TestMetricsUnaryInterceptor_WithErrorInProto` | Тестирует разные типы protobuf сообщений |
| `TestMetricsStreamInterceptor_DifferentStatusCodes` | Тестирует разные статус коды для stream |
| `TestMetricsUnaryInterceptor_WithSpanInContext` | Тестирует интерцептор со спаном в контексте |
| `TestMetricsUnaryInterceptor_MeasuresDuration` | Тестирует измерение длительности |
| `TestMetricsStreamInterceptor_MeasuresDuration` | Тестирует измерение длительности для stream |

**Тесты с `t.Parallel()` (не используют глобальное состояние):**

| Тест | Описание |
|------|----------|
| `TestGetMessageSize_ProtoMessage` | Тестирует getMessageSize с protobuf сообщениями |
| `TestGetMessageSize_NonProtoMessage` | Тестирует getMessageSize с non-protobuf сообщениями |
| `TestGetMessageSize_WithNilProto` | Тестирует getMessageSize с nil proto message |

---

## grpc/middleware/logging_test.go

### Причина: общие переменные между родительским и под-тестами

В некоторых тестах родительские тесты имеют `t.Parallel()` и содержат под-тесты, которые также имеют `t.Parallel()`. Под-тесты используют общие переменные (`logAttrs`, `logger`, `interceptor`, `info`, `handler`), которые создаются в родительском тесте. При параллельном выполнении под-тестов они модифицируют эти переменные одновременно, что вызывает race conditions.

**Тесты без `t.Parallel()` из-за общих переменных:**

| Тест | Общие переменные |
|------|------------------|
| `TestRecoveryInterceptor_CatchesPanic` | `logAttrs`, `logger`, `interceptor`, `info` |
| `TestRecoveryStreamInterceptor_CatchesPanic` | `logAttrs`, `logger`, `interceptor`, `info`, `ss` |
| `TestLoggingStreamInterceptor_StreamTypes` | `logAttrs`, `logger`, `interceptor` |
| `TestLoggingInterceptor_WithPanic` | `logAttrs`, `logger`, `recovery`, `logging`, `info`, `panicMsg`, `handler` |
| `TestLoggingInterceptor_DifferentErrorCodes` | `testCases` (используется для итерации по под-тестам) |

---

## metrics/metrics_test.go

### Причина: глобальное состояние Prometheus

Тест `TestNewHttpServer/registers_metrics_endpoint` использует `promhttp.Handler()` для проверки endpoint `/metrics`. Когда тесты запускаются параллельно с другими тестами, которые вызывают `InitPrometheus()` (например, `TestMetrics_Start` или `TestInitDefault`), глобальное состояние Prometheus может быть инициализировано несколько раз, что вызывает race conditions.

Функция `InitPrometheus()` устанавливает глобальное состояние через:
- `otel.SetMeterProvider(provider)` — устанавливает глобальный провайдер метрик
- `runtime.Start()` — запускает instrumentation рантайма

При параллельном выполнении тестов эти глобальные настройки могут конфликтовать, что приводит к нестабильному поведению тестов.

**Тесты без `t.Parallel()` из-за глобального состояния Prometheus:**

| Тест | Описание |
|------|----------|
| `TestNewHttpServer` | Родительский тест, который содержит под-тест `registers_metrics_endpoint`, использующий глобальный `promhttp.Handler()` |

**Тесты с `t.Parallel()` (под-тесты, не использующие глобальное состояние):**

| Тест | Описание |
|------|----------|
| `TestNewHttpServer/sets_correct_read_timeout` | Проверяет настройку таймаута чтения |
| `TestNewHttpServer/uses_default_read_timeout_when_not_specified` | Проверяет использование значения по умолчанию для таймаута |

---

## Итоговое правило

Тест **не должен** использовать `t.Parallel()`, если он:

1. Вызывает `t.Setenv()` — Go паникует в рантайме
2. Вызывает `t.Chdir()` — Go паникует в рантайме
3. Напрямую вызывает `os.Setenv()` / `os.Unsetenv()` / `os.Chdir()` — нет паники, но возможны гонки данных
4. Рассчитывает на чистое окружение, которое может быть загрязнено другими тестами (например, через загрузку `.env`-файлов)
5. **Имеет общие переменные с под-тестами** — под-тесты могут модифицировать эти переменные параллельно, создавая гонки данных
6. **Является родительским тестом с под-тестами** — если родительский тест имеет `t.Parallel()`, под-тесты должны выполняться последовательно, чтобы избежать гонок данных
7. **Устанавливает глобальное состояние OpenTelemetry** — вызовы `otel.SetTextMapPropagator()`, `otel.SetTracerProvider()`, `otel.SetMeterProvider()` устанавливают глобальное состояние для всего процесса, что вызывает race conditions при параллельном выполнении тестов
