# PostgreSQL адаптер на базе Kafka

Адаптер для работы с Apache Kafka, построенный на библиотеке [segmentio/kafka-go](https://github.com/segmentio/kafka-go).

## Возможности

- Подключение к Kafka с настраиваемыми параметрами
- Публикация сообщений в Kafka
- Потребление сообщений из Kafka с поддержкой групп потребителей
- Поддержка различных кодировщиков сообщений (JSON, текст)
- Трейсинг сообщений через OpenTelemetry
- Поддержка балансировки сообщений между партициями
- Обработка ошибок с возможностью повтора

## Использование

### Подключение к Kafka

```go
cfg := kafka.Config{
    Brokers: []string{"localhost:9092"},
    GroupID: "my-consumer-group",
}

dialer := kafka.NewDialer(cfg, nil)
defer dialer.Close()
```

### Публикация сообщений

```go
// Создаем publisher
pub := kafka.NewPublisher(dialer, kafka.PublisherConfig{
    Encoder:  encoders.JSON{},
    Balancer: &kafka.LeastBytes{},
})
defer pub.Close()

// Отправляем сообщение
msg := queue.Message{
    Topic:   "my-topic",
    Body:    map[string]string{"key": "value"},
    Headers: map[string]string{"header1": "value1"},
}

err := pub.Publish(context.Background(), msg)
```

### Потребление сообщений

```go
// Создаем subscriber
sub := kafka.NewSubscriber(dialer, "my-topic", kafka.SubscriberConfig{
    Name:         "my-subscriber",
    PrefetchCount: 1,
    MaxTryNum:     3,
    Backoff:       5 * time.Second,
})

// Запускаем обработку сообщений
go sub.Listen(func(ctx context.Context, msg queue.Delivery) (bool, error) {
    // Обрабатываем сообщение
    fmt.Println("Received:", string(msg.Body))

    // Возвращаем false для успешной обработки
    // Возвращаем true для повторной попытки
    return false, nil
})

// Закрываем subscriber при завершении
defer sub.Close()
```

### Использование с группой потребителей

```go
cfg := kafka.Config{
    Brokers: []string{"localhost:9092"},
    GroupID: "my-group", // Все потребители с одинаковым GroupID будут разделять сообщения
}

dialer := kafka.NewDialer(cfg, nil)

// Несколько потребителей с одной группой будут разделять нагрузку
sub1 := kafka.NewSubscriber(dialer, "topic", kafka.SubscriberConfig{Name: "consumer-1"})
sub2 := kafka.NewSubscriber(dialer, "topic", kafka.SubscriberConfig{Name: "consumer-2"})
```

## Конфигурация

### Config (подключение к Kafka)

```go
type Config struct {
    Brokers []string // список брокеров Kafka (обязательно)
    GroupID string   // идентификатор группы потребителей
}
```

### PublisherConfig (публикация)

```go
type PublisherConfig struct {
    Balancer kafka.Balancer // стратегия балансировки (по умолчанию: &kafka.LeastBytes{})
    Encoder  queue.Encoder  // кодировщик сообщений (по умолчанию: JSON)
}
```

### SubscriberConfig (потребление)

```go
type SubscriberConfig struct {
    Name         string        // имя потребителя (для логирования)
    PrefetchCount int          // максимальное количество одновременно обрабатываемых сообщений
    MaxTryNum    int           // максимальное количество попыток обработки (-1 для бесконечных)
    Backoff      time.Duration // время ожидания между попытками
}
```

## Стратегии балансировки

Kafka поддерживает различные стратегии балансировки сообщений между партициями:

```go
// Наименьшее количество байт (по умолчанию)
&kafka.LeastBytes{}

// Хэш по ключу сообщения
&kafka.Hash{}

// КруговаяRobin
&kafka.RoundRobin{}

// Случайная
&kafka.Random{}
```

## Обработка ошибок

### Retryable ошибки

Если обработчик возвращает `true` и ошибку, сообщение будет повторно обработано:

```go
handler := func(ctx context.Context, msg queue.Delivery) (bool, error) {
    // Попытка обработки
    if err := processMessage(msg); err != nil {
        if isTemporaryError(err) {
            return true, err // Повторить
        }
    }
    return false, nil
}
```

### Non-retryable ошибки

Если обработчик возвращает `false`, сообщение будет подтверждено (committed):

```go
handler := func(ctx context.Context, msg queue.Delivery) (bool, error) {
    if err := processMessage(msg); err != nil {
        return false, err // Не повторять, подтвердить сообщение
    }
    return false, nil
}
```

## Трейсинг

Адаптер автоматически внедряет контекст OpenTelemetry в заголовки сообщений Kafka. Это позволяет отслеживать распределенные транзакции через Kafka.

При публикации и потреблении сообщений создаются спаны с атрибутами:
- `topic`: имя темы Kafka
- `partition`: номер партиции
- `offset`: смещение сообщения
- `body_size`: размер тела сообщения
- `headers_count`: количество заголовков

## Тестирование

Для запуска интеграционных тестов необходим запущенный Kafka:

```bash
# Запуск Kafka через Docker
docker run -d --name kafka \
  -p 9092:9092 \
  -e KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181 \
  -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 \
  confluentinc/cp-kafka:latest

# Запуск тестов
go test ./queue/kafka

# Пропуск интеграционных тестов
go test -short ./queue/kafka
```

## Переменные окружения

Конфигурация через переменные окружения:

```bash
# Kafka брокеры
export KAFKA_BROKERS=localhost:9092,localhost:9093

# Группа потребителей
export KAFKA_GROUP_ID=my-consumer-group
```

## Сравнение с RabbitMQ

### Преимущества Kafka:
- Высокая производительность при больших объемах данных
- Поддержка репликации и отказоустойчивости
- Хранение сообщений на диске
- Поддержка ретроспективного чтения сообщений

### Когда использовать RabbitMQ вместо Kafka:
- Необходимость сложной маршрутизации сообщений
- Низкая задержка доставки (latency) критична
- Меньшее количество сообщений и сложные паттерны обмена

## Примечания

- Kafka использует концепцию тем и партиций
- Сообщения в Kafka упорядочены внутри партиции
- Группы потребителей позволяют разделять нагрузку между несколькими экземплярами
- Offset'ы сообщений управляются автоматически группой потребителей
- Все операции поддерживают context.Context для отмены и таймаутов
- Все сообщения трейсируются через OpenTelemetry
