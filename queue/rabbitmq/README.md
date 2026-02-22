# queue/rabbitmq

Адаптер для работы с RabbitMQ на базе библиотеки [amqp091-go](https://github.com/rabbitmq/amqp091-go).

Поддерживает:
- публикацию сообщений с трассировкой OpenTelemetry;
- подписку на одну очередь (`Subscriber`) с DLX-повторами;
- подписку на несколько очередей одновременно (`MultiQueueSubscriber`);
- декларирование топологии через JSON (`Definitions`) для CI/CD и интеграционных тестов.

## Установка зависимости

```
github.com/pure-golang/adapters/queue/rabbitmq
```

## Конфигурация соединения

### Параметры Dialer

| Параметр | Тип | Описание |
|---|---|---|
| `uri` | `string` | AMQP URI (например, `amqp://guest:guest@localhost:5672/`) |
| `RetryPolicy` | `RetryPolicy` | Политика переподключения. По умолчанию `NewDefaultMaxInterval()` |
| `Logger` | `*slog.Logger` | Логгер. По умолчанию `slog.Default()` |

### Подключение

```go
dialer := rabbitmq.NewDialer("amqp://guest:guest@localhost:5672/", nil)
if err := dialer.Connect(); err != nil {
    log.Fatal(err)
}
defer dialer.Close()
```

## Публикация сообщений

### Параметры PublisherConfig

| Параметр | Тип | Описание |
|---|---|---|
| `Exchange` | `string` | Имя обменника |
| `RoutingKey` | `string` | Routing key по умолчанию |
| `DeliveryMode` | `DeliveryMode` | `Transient` или `Persistent` (по умолчанию `Persistent`) |
| `Encoder` | `queue.Encoder` | Кодировщик сообщений. По умолчанию JSON |
| `MessageTTL` | `time.Duration` | TTL для всех сообщений (точность до миллисекунд) |

```go
publisher := rabbitmq.NewPublisher(dialer, rabbitmq.PublisherConfig{
    Exchange:    "",
    RoutingKey:  "my-queue",
    DeliveryMode: rabbitmq.Persistent,
})

err := publisher.Publish(ctx, queue.Message{
    Value: MyEvent{ID: 42, Name: "test"},
})
```

## Подписка на одну очередь

`Subscriber` читает сообщения из одной очереди. Повторные попытки реализованы
через Dead Letter Exchange: при ошибке сообщение отправляется в retry-очередь
(с `x-message-ttl`), откуда RabbitMQ возвращает его в основную очередь по истечении TTL.
Счётчик попыток хранится в заголовке `x-death` (стандартный механизм RabbitMQ).

### Параметры SubscriberOptions

| Параметр | Тип | По умолчанию | Описание |
|---|---|---|---|
| `Name` | `string` | UUID | Consumer tag AMQP |
| `PrefetchCount` | `int` | `1` | Qos prefetch count |
| `MaxRetries` | `int` | `0` | Максимальное число попыток перед DLQ |
| `RetryQueueName` | `string` | `queueName + ".retry"` | Очередь для повторных попыток |
| `MessageTimeout` | `time.Duration` | `0` (без таймаута) | Таймаут обработки одного сообщения |

```go
subscriber := rabbitmq.NewSubscriber(dialer, "my-queue", rabbitmq.SubscriberOptions{
    MaxRetries:     3,
    MessageTimeout: 30 * time.Second,
})

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

subscriber.Listen(ctx, func(ctx context.Context, d queue.Delivery) (bool, error) {
    var event MyEvent
    if err := json.Unmarshal(d.Body, &event); err != nil {
        return true, err // true = повторить
    }
    // обработка события...
    return false, nil // false = успех
})
```

## Подписка на несколько очередей

`MultiQueueSubscriber` читает из нескольких очередей на одном AMQP-канале.
Единая горутина обрабатывает сообщения из всех очередей последовательно.
При `PrefetchCount=1` гарантируется не более одного сообщения в обработке — это
оптимально для CPU-интенсивных задач.

### Параметры MultiQueueOptions

| Параметр | Тип | По умолчанию | Описание |
|---|---|---|---|
| `PrefetchCount` | `int` | `1` | Qos prefetch (global=true для всего канала) |
| `MaxRetries` | `int` | `0` | Максимальное число попыток перед DLQ |
| `MessageTimeout` | `time.Duration` | `0` | Таймаут обработки одного сообщения |
| `ReconnectDelay` | `time.Duration` | `5s` | Пауза между попытками переподключения |

```go
sub := rabbitmq.NewMultiQueueSubscriber(dialer, rabbitmq.MultiQueueOptions{
    PrefetchCount:  1,
    MaxRetries:     3,
    MessageTimeout: 30 * time.Second,
})

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

handler := func(ctx context.Context, d queue.Delivery) (bool, error) {
    // обработка сообщения...
    return false, nil
}

if err := sub.Listen(ctx,
    rabbitmq.QueueHandler{QueueName: "queue-a", Handler: handler},
    rabbitmq.QueueHandler{QueueName: "queue-b", Handler: handler},
); err != nil {
    log.Println(err)
}
```

## Топология (Definitions)

`Definitions` отражает формат JSON management API RabbitMQ. Используется для:

- генерации `rabbitmq_definitions.json` для `rabbitmqctl import_definitions` / `load_definitions` в CI/CD;
- прямого применения топологии через AMQP в интеграционных тестах.

### Пример DLX-топологии для retry-паттерна

```go
prefix := "my-service"
defs := rabbitmq.Definitions{
    Vhosts: []rabbitmq.VhostDef{{Name: "/"}},
    Exchanges: []rabbitmq.ExchangeDef{
        {Name: prefix + ".dlx", Vhost: "/", Type: "direct", Durable: true},
    },
    Queues: []rabbitmq.QueueDef{
        {
            Name: prefix + ".main", Vhost: "/", Durable: true,
            Arguments: map[string]any{
                "x-dead-letter-exchange":    "",
                "x-dead-letter-routing-key": prefix + ".dlq",
            },
        },
        {
            Name: prefix + ".retry", Vhost: "/", Durable: true,
            Arguments: map[string]any{
                "x-dead-letter-exchange":    "",
                "x-dead-letter-routing-key": prefix + ".main",
                "x-message-ttl":             int32(30_000), // 30 секунд
            },
        },
        {Name: prefix + ".dlq", Vhost: "/", Durable: true},
    },
    Bindings: []rabbitmq.BindingDef{
        {
            Source: prefix + ".dlx", Destination: prefix + ".main",
            Vhost: "/", DestinationType: "queue",
            RoutingKey: prefix + ".main",
        },
    },
}
```

### Сохранение в JSON для CI/CD

```go
data, err := defs.JSON()
if err != nil {
    log.Fatal(err)
}
os.WriteFile("rabbitmq_definitions.json", data, 0o644)
```

### Применение через AMQP в тестах

```go
ch, err := dialer.Channel()
if err != nil {
    t.Fatal(err)
}
defer ch.Close()

if err := defs.applyDefinitions(ch); err != nil {
    t.Fatal(err)
}
```

### Загрузка в docker-compose

```yaml
rabbitmq:
  image: rabbitmq:management-alpine
  volumes:
    - ./rabbitmq_definitions.json:/etc/rabbitmq/definitions.json:ro
  environment:
    RABBITMQ_SERVER_ADDITIONAL_ERL_ARGS: >
      -rabbitmq_management load_definitions "/etc/rabbitmq/definitions.json"
```

## Обработка ошибок

Функция-обработчик `queue.Handler` возвращает `(bool, error)`:

- `(false, nil)` — успех, сообщение подтверждается (`Ack`);
- `(true, err)` — временная ошибка, сообщение отправляется в retry-очередь;
- если счётчик `x-death` достиг `MaxRetries` — сообщение отправляется в DLQ (`Nack requeue=false`).

## Трассировка

Адаптер автоматически извлекает контекст трассировки из заголовков AMQP при получении
и инжектирует при публикации (стандарт W3C TraceContext через `otel.GetTextMapPropagator()`).
