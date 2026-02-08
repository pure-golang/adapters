package kafka

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pure-golang/adapters/queue"
	"github.com/pure-golang/adapters/queue/encoders"
)

func (s *KafkaSuite) TestSubscriber_Listen() {
	ctx := context.Background()

	cfg := Config{
		Brokers: s.brokers,
	}
	dialer := NewDialer(cfg, nil)

	s.T().Cleanup(func() {
		require.NoError(s.T(), dialer.Close())
	})

	// Создаем publisher для отправки тестовых сообщений
	pub := NewPublisher(dialer, PublisherConfig{
		Encoder: encoders.Text{},
	})

	s.T().Cleanup(func() {
		require.NoError(s.T(), pub.Close())
	})

	// Создаем уникальную тему для этого теста, чтобы избежать конфликтов с другими тестами
	uniqueTopic := "test-subscriber-listen-" + uuid.NewString()

	// Создаем тему для теста используя то же подключение, что и publisher
	conn, err := kafkago.Dial("tcp", s.brokers[0])
	require.NoError(s.T(), err)
	s.T().Cleanup(func() {
		if err := conn.Close(); err != nil {
			s.T().Logf("failed to close conn: %v", err)
		}
	})

	err = conn.CreateTopics(kafkago.TopicConfig{
		Topic:             uniqueTopic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil {
		s.T().Logf("Warning: failed to create topic: %v", err)
	}

	// Создаем subscriber
	var wg sync.WaitGroup
	wg.Add(1)

	var messageMu sync.Mutex
	var receivedMessage queue.Delivery
	messageReceived := make(chan struct{}, 1)

	handler := func(ctx context.Context, msg queue.Delivery) (bool, error) {
		messageMu.Lock()
		receivedMessage = msg
		messageMu.Unlock()
		messageReceived <- struct{}{}
		return false, nil
	}

	sub := NewSubscriber(dialer, uniqueTopic, SubscriberConfig{
		Name: "test-subscriber-" + uuid.NewString(),
	})

	// Запускаем subscriber в отдельной горутине
	go func() {
		defer wg.Done()
		sub.Listen(handler)
	}()

	s.T().Cleanup(func() {
		require.NoError(s.T(), sub.Close())
		wg.Wait()
	})

	// Ждем немного, чтобы subscriber подключился и joined the consumer group
	// Увеличили время ожидания для более надежной работы в CI
	time.Sleep(5 * time.Second)

	// Проверяем что publisher может фактически записать в новую тему
	// Это заставит publisher обновить метаданные и убедится, что тема готова
	s.T().Logf("Pinging topic to verify publisher metadata is ready...")
	for i := 0; i < 20; i++ {
		err = pub.Publish(ctx, queue.Message{Topic: uniqueTopic, Body: "ping"})
		if err == nil {
			s.T().Logf("Publisher successfully pinged topic after %d attempts", i+1)
			break
		}
		s.T().Logf("Publisher ping attempt %d/20 failed: %v", i+1, err)
		if i < 19 {
			time.Sleep(500 * time.Millisecond)
		}
	}
	// Если ping не удался, продолжаем тест - возможно ошибка будет более информативной на следующем шаге

	// Отправляем тестовое сообщение
	testMsg := queue.Message{
		Topic: uniqueTopic,
		Body:  "test message",
	}

	err = pub.Publish(ctx, testMsg)
	require.NoError(s.T(), err)

	// Ждем получения сообщения
	select {
	case <-messageReceived:
		messageMu.Lock()
		assert.NotNil(s.T(), receivedMessage.Body)
		assert.NotEmpty(s.T(), receivedMessage.Body)
		messageMu.Unlock()
	case <-time.After(20 * time.Second):
		s.T().Fatal("Timeout waiting for message")
	}
}

func (s *KafkaSuite) TestSubscriber_WithRetryableError() {
	ctx := context.Background()

	cfg := Config{
		Brokers: s.brokers,
	}
	dialer := NewDialer(cfg, nil)

	s.T().Cleanup(func() {
		require.NoError(s.T(), dialer.Close())
	})

	// Создаем publisher
	pub := NewPublisher(dialer, PublisherConfig{
		Encoder: encoders.JSON{},
	})

	s.T().Cleanup(func() {
		require.NoError(s.T(), pub.Close())
	})

	// Создаем subscriber с коротким backoff для тестов
	var attemptCount int
	var wg sync.WaitGroup
	wg.Add(1)

	handler := func(ctx context.Context, msg queue.Delivery) (bool, error) {
		attemptCount++
		if attemptCount < 2 {
			// Первая попытка возвращает retryable ошибку
			return true, errors.New("temporary error")
		}
		// Вторая попытка успешна
		return false, nil
	}

	sub := NewSubscriber(dialer, s.topic, SubscriberConfig{
		Name:    "test-subscriber-retry-" + uuid.NewString(),
		Backoff: 100 * time.Millisecond,
	})

	// Запускаем subscriber в отдельной горутине
	go func() {
		defer wg.Done()
		sub.Listen(handler)
	}()

	s.T().Cleanup(func() {
		require.NoError(s.T(), sub.Close())
		wg.Wait()
	})

	// Ждем подключения
	time.Sleep(2 * time.Second)

	// Отправляем тестовое сообщение
	testMsg := queue.Message{
		Topic: s.topic,
		Body:  "test retry",
	}

	err := pub.Publish(ctx, testMsg)
	require.NoError(s.T(), err)

	// Ждем завершения обработки
	time.Sleep(1 * time.Second)
}

func (s *KafkaSuite) TestSubscriber_WithNonRetryableError() {
	ctx := context.Background()

	cfg := Config{
		Brokers: s.brokers,
	}
	dialer := NewDialer(cfg, nil)

	s.T().Cleanup(func() {
		require.NoError(s.T(), dialer.Close())
	})

	// Создаем publisher
	pub := NewPublisher(dialer, PublisherConfig{
		Encoder: encoders.JSON{},
	})

	s.T().Cleanup(func() {
		require.NoError(s.T(), pub.Close())
	})

	// Создаем subscriber
	var wg sync.WaitGroup
	wg.Add(1)

	handler := func(ctx context.Context, msg queue.Delivery) (bool, error) {
		// Возвращаем non-retryable ошибку
		return false, errors.New("permanent error")
	}

	sub := NewSubscriber(dialer, s.topic, SubscriberConfig{
		Name: "test-subscriber-non-retry-" + uuid.NewString(),
	})

	// Запускаем subscriber в отдельной горутине
	go func() {
		defer wg.Done()
		sub.Listen(handler)
	}()

	s.T().Cleanup(func() {
		require.NoError(s.T(), sub.Close())
		wg.Wait()
	})

	// Ждем подключения
	time.Sleep(2 * time.Second)

	// Отправляем тестовое сообщение
	testMsg := queue.Message{
		Topic: s.topic,
		Body:  "test non-retry",
	}

	err := pub.Publish(ctx, testMsg)
	require.NoError(s.T(), err)

	// Ждем обработки
	time.Sleep(1 * time.Second)
}

func (s *KafkaSuite) TestSubscriber_DefaultConfig() {
	cfg := Config{
		Brokers: s.brokers,
	}
	dialer := NewDialer(cfg, nil)

	// Создаем subscriber с дефолтным конфигом
	sub := NewDefaultSubscriber(dialer, s.topic)

	assert.NotNil(s.T(), sub)
	require.NoError(s.T(), sub.Close())
	require.NoError(s.T(), dialer.Close())
}

func (s *KafkaSuite) TestSubscriber_CustomConfig() {
	cfg := Config{
		Brokers: s.brokers,
	}
	dialer := NewDialer(cfg, nil)

	// Создаем subscriber с кастомным конфигом
	sub := NewSubscriber(dialer, s.topic, SubscriberConfig{
		Name:          "custom-sub",
		PrefetchCount: 5,
		MaxTryNum:     10,
		Backoff:       2 * time.Second,
	})

	assert.NotNil(s.T(), sub)
	require.NoError(s.T(), sub.Close())
	require.NoError(s.T(), dialer.Close())
}

func (s *KafkaSuite) TestSubscriber_Close() {
	cfg := Config{
		Brokers: s.brokers,
	}
	dialer := NewDialer(cfg, nil)

	// Создаем subscriber
	sub := NewSubscriber(dialer, s.topic, SubscriberConfig{
		Name: "test-close",
	})

	// Закрываем subscriber
	err := sub.Close()
	assert.NoError(s.T(), err)

	// Повторный Close не должен вызывать ошибку
	err = sub.Close()
	assert.NoError(s.T(), err)

	require.NoError(s.T(), dialer.Close())
}
