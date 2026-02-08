package kafka

import (
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("github.com/pure-golang/adapters/queue/kafka")

// headersCarrier реализует propagation.TextMapCarrier для передачи трейсов через заголовки Kafka
type headersCarrier map[string]string

// Get возвращает значение заголовка по ключу
func (c headersCarrier) Get(key string) string {
	return c[key]
}

// Set устанавливает значение заголовка
func (c headersCarrier) Set(key, value string) {
	c[key] = value
}

// Keys возвращает список всех ключей заголовков
func (c headersCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
