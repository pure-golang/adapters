package rabbitmq

import (
	amqp "github.com/rabbitmq/amqp091-go"

	"go.opentelemetry.io/otel"

	"fmt"
)

var tracer = otel.Tracer("github.com/pure-golang/adapters/queue/rabbitmq")

// tableCarrier implements propagation.TextMapCarrier to transfer traces by AMQP protocol
type tableCarrier amqp.Table

func (c tableCarrier) Get(key string) string {
	return fmt.Sprintf("%v", c[key])
}

func (c tableCarrier) Set(key, value string) {
	c[key] = value
}

func (c tableCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
