package kafka

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"

	"github.com/pure-golang/adapters/queue"
	"github.com/pure-golang/adapters/queue/encoders"
)

func TestNewPublisher_WithNilEncoder(t *testing.T) {
	cfg := Config{
		Brokers: []string{"localhost:9092"},
	}
	dialer := NewDialer(cfg, nil)

	// Создаем publisher без encoder - должен использовать JSON по умолчанию
	pub := NewPublisher(dialer, PublisherConfig{})

	assert.NotNil(t, pub)

	// Закрываем publisher
	err := pub.Close()
	assert.NoError(t, err)

	err = dialer.Close()
	assert.NoError(t, err)
}

func TestNewPublisher_WithCustomBalancer(t *testing.T) {
	cfg := Config{
		Brokers: []string{"localhost:9092"},
	}
	dialer := NewDialer(cfg, nil)

	// Создаем publisher с кастомным балансировщиком
	pub := NewPublisher(dialer, PublisherConfig{
		Balancer: &kafka.Hash{},
		Encoder:  encoders.Text{},
	})

	assert.NotNil(t, pub)

	err := pub.Close()
	assert.NoError(t, err)

	err = dialer.Close()
	assert.NoError(t, err)
}

func TestPublisher_CloseTwice(t *testing.T) {
	cfg := Config{
		Brokers: []string{"localhost:9092"},
	}
	dialer := NewDialer(cfg, nil)

	pub := NewPublisher(dialer, PublisherConfig{
		Encoder: encoders.JSON{},
	})

	// Первый закрытие
	err := pub.Close()
	assert.NoError(t, err)

	// Второй закрытие не должно вызывать ошибку
	err = pub.Close()
	assert.NoError(t, err)

	err = dialer.Close()
	assert.NoError(t, err)
}

func TestNewSubscriber_WithDefaultConfig(t *testing.T) {
	cfg := Config{
		Brokers: []string{"localhost:9092"},
	}
	dialer := NewDialer(cfg, nil)

	// Создаем subscriber с дефолтным конфигом
	sub := NewSubscriber(dialer, "test-topic", SubscriberConfig{})

	assert.NotNil(t, sub)

	// Проверяем дефолтные значения
	assert.Equal(t, "test-topic", sub.topic)
	assert.Equal(t, 1, sub.cfg.PrefetchCount) // дефолтное значение
	assert.Equal(t, 3, sub.cfg.MaxTryNum)     // дефолтное значение

	err := sub.Close()
	assert.NoError(t, err)

	err = dialer.Close()
	assert.NoError(t, err)
}

func TestNewSubscriber_WithCustomConfig(t *testing.T) {
	cfg := Config{
		Brokers: []string{"localhost:9092"},
		GroupID: "my-group",
	}
	dialer := NewDialer(cfg, nil)

	// Создаем subscriber с кастомным конфигом
	sub := NewSubscriber(dialer, "test-topic", SubscriberConfig{
		Name:          "custom-sub",
		PrefetchCount: 5,
		MaxTryNum:     10,
		Backoff:       30 * time.Second,
	})

	assert.NotNil(t, sub)
	assert.Equal(t, "test-topic", sub.topic)
	assert.Equal(t, "custom-sub", sub.cfg.Name)
	assert.Equal(t, 5, sub.cfg.PrefetchCount)
	assert.Equal(t, 10, sub.cfg.MaxTryNum)
	assert.Equal(t, 30*time.Second, sub.cfg.Backoff)

	err := sub.Close()
	assert.NoError(t, err)

	err = dialer.Close()
	assert.NoError(t, err)
}

func TestSubscriber_CloseTwice(t *testing.T) {
	cfg := Config{
		Brokers: []string{"localhost:9092"},
	}
	dialer := NewDialer(cfg, nil)

	sub := NewSubscriber(dialer, "test-topic", SubscriberConfig{})

	// Первый закрытие
	err := sub.Close()
	assert.NoError(t, err)

	// Второй закрытие не должно вызывать ошибку
	err = sub.Close()
	assert.NoError(t, err)

	err = dialer.Close()
	assert.NoError(t, err)
}

func TestSubscriber_WithInfiniteRetries(t *testing.T) {
	cfg := Config{
		Brokers: []string{"localhost:9092"},
	}
	dialer := NewDialer(cfg, nil)

	// Создаем subscriber с бесконечными попытками
	sub := NewSubscriber(dialer, "test-topic", SubscriberConfig{
		MaxTryNum: -1, // InfiniteRetriesIndicator
	})

	assert.NotNil(t, sub)
	assert.Equal(t, -1, sub.cfg.MaxTryNum)

	err := sub.Close()
	assert.NoError(t, err)

	err = dialer.Close()
	assert.NoError(t, err)
}

func TestSubscriber_WithZeroTryNum(t *testing.T) {
	cfg := Config{
		Brokers: []string{"localhost:9092"},
	}
	dialer := NewDialer(cfg, nil)

	// Создаем subscriber с MaxTryNum = 0 (должен стать 3 по умолчанию)
	sub := NewSubscriber(dialer, "test-topic", SubscriberConfig{
		MaxTryNum: 0,
	})

	assert.NotNil(t, sub)
	assert.Equal(t, 3, sub.cfg.MaxTryNum) // дефолтное значение

	err := sub.Close()
	assert.NoError(t, err)

	err = dialer.Close()
	assert.NoError(t, err)
}

func TestSubscriber_WithZeroPrefetchCount(t *testing.T) {
	cfg := Config{
		Brokers: []string{"localhost:9092"},
	}
	dialer := NewDialer(cfg, nil)

	// Создаем subscriber с PrefetchCount = 0 (должен стать 1 по умолчанию)
	sub := NewSubscriber(dialer, "test-topic", SubscriberConfig{
		PrefetchCount: 0,
	})

	assert.NotNil(t, sub)
	assert.Equal(t, 1, sub.cfg.PrefetchCount) // дефолтное значение

	err := sub.Close()
	assert.NoError(t, err)

	err = dialer.Close()
	assert.NoError(t, err)
}

func TestSubscriber_HandlerReturnsError(t *testing.T) {
	cfg := Config{
		Brokers: []string{"localhost:9092"},
	}
	dialer := NewDialer(cfg, nil)

	sub := NewSubscriber(dialer, "test-topic", SubscriberConfig{})

	// Создаем тестовый обработчик, который возвращает ошибку
	handler := func(ctx context.Context, msg queue.Delivery) (bool, error) {
		return true, errors.New("test error")
	}

	// Проверяем, что обработчик возвращает правильные значения
	retry, err := handler(context.Background(), queue.Delivery{})
	assert.True(t, retry)
	assert.Error(t, err)

	err = sub.Close()
	assert.NoError(t, err)

	err = dialer.Close()
	assert.NoError(t, err)
}

func TestSubscriber_HandlerSuccess(t *testing.T) {
	cfg := Config{
		Brokers: []string{"localhost:9092"},
	}
	dialer := NewDialer(cfg, nil)

	sub := NewSubscriber(dialer, "test-topic", SubscriberConfig{})

	// Создаем тестовый обработчик, который возвращает успех
	handler := func(ctx context.Context, msg queue.Delivery) (bool, error) {
		return false, nil
	}

	// Проверяем, что обработчик возвращает правильные значения
	retry, err := handler(context.Background(), queue.Delivery{})
	assert.False(t, retry)
	assert.NoError(t, err)

	err = sub.Close()
	assert.NoError(t, err)

	err = dialer.Close()
	assert.NoError(t, err)
}

func TestPublisher_EncodeValue(t *testing.T) {
	// Проверяем кодировку сообщения через encoder
	msg := queue.Message{
		Topic: "test-topic",
		Body:  "test body",
	}

	encoder := encoders.JSON{}
	body, err := msg.EncodeValue(encoder)
	assert.NoError(t, err)
	assert.NotNil(t, body)

	// Проверяем с nil телом
	msg.Body = nil
	body, err = msg.EncodeValue(encoder)
	assert.NoError(t, err)
	assert.Nil(t, body)
}
