package queue

import (
	"context"
	"io"
	"time"
)

// Publisher sends messages to topic of message broker.
type Publisher interface {
	Publish(ctx context.Context, msgs ...Message) error
}

// Subscriber listens messages from topic of message broker.
type Subscriber interface {
	Listen(h Handler)
	io.Closer
}

// Handler returns bool=true if error is retryable.
type Handler func(ctx context.Context, msg Delivery) (bool, error)

// Encoder converts interface{} to []byte.
type Encoder interface {
	Encode(i any) ([]byte, error)
	ContentType() string
}

// Message is used to publish messages to message broker.
type Message struct {
	Topic   string
	Headers map[string]string
	Body    any
	TTL     time.Duration
}

// EncodeValue converts Body to []byte using Encoder if Body != nil.
func (m *Message) EncodeValue(enc Encoder) ([]byte, error) {
	if m.Body == nil {
		return nil, nil
	}
	return enc.Encode(m.Body)
}

// Delivery is used to consume messages from message broker.
type Delivery struct {
	Headers map[string]string
	Body    []byte
}
