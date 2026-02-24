package kafka_test

import (
	"context"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pure-golang/adapters/queue"
	"github.com/pure-golang/adapters/queue/encoders"
	"github.com/pure-golang/adapters/queue/kafka"
)

func (s *KafkaSuite) TestPublisher_Publish() {
	ctx := context.Background()

	pub := s.createPublisher(encoders.JSON{})

	// Отправляем сообщение
	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	testMsg := TestData{
		Name:  "test",
		Value: 42,
	}

	msg := queue.Message{
		Topic: s.topic,
		Body:  testMsg,
	}

	err := pub.Publish(ctx, msg)
	require.NoError(s.T(), err)

	// Читаем сообщение из Kafka
	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers: s.brokers,
		Topic:   s.topic,
		GroupID: "test-consumer-" + s.topic,
	})
	s.T().Cleanup(func() {
		if err := reader.Close(); err != nil {
			s.T().Logf("failed to close reader: %v", err)
		}
	})

	// Ждем сообщения с таймаутом
	kafkaMsg, err := reader.ReadMessage(ctx)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), s.topic, kafkaMsg.Topic)
	assert.NotEmpty(s.T(), kafkaMsg.Value)
}

func (s *KafkaSuite) TestPublisher_PublishMultiple() {
	ctx := context.Background()

	pub := s.createPublisher(encoders.JSON{})

	// Отправляем несколько сообщений
	messages := []queue.Message{
		{Topic: s.topic, Body: "message1"},
		{Topic: s.topic, Body: "message2"},
		{Topic: s.topic, Body: "message3"},
	}

	err := pub.Publish(ctx, messages...)
	require.NoError(s.T(), err)
}

func (s *KafkaSuite) TestPublisher_WithHeaders() {
	ctx := context.Background()

	pub := s.createPublisher(encoders.JSON{})

	// Отправляем сообщение с заголовками
	msg := queue.Message{
		Topic:   s.topic,
		Body:    "test message",
		Headers: map[string]string{"key1": "value1", "key2": "value2"},
	}

	err := pub.Publish(ctx, msg)
	require.NoError(s.T(), err)
}

func (s *KafkaSuite) TestPublisher_WithTextEncoder() {
	ctx := context.Background()

	pub := s.createPublisher(encoders.Text{})

	msg := queue.Message{
		Topic: s.topic,
		Body:  "plain text message",
	}

	err := pub.Publish(ctx, msg)
	require.NoError(s.T(), err)
}

func (s *KafkaSuite) TestPublisher_WhenClosed() {
	ctx := context.Background()

	dialer := s.createDialer()
	pub := kafka.NewPublisher(dialer, kafka.PublisherConfig{
		Encoder: encoders.JSON{},
	})

	// Закрываем publisher
	err := pub.Close()
	require.NoError(s.T(), err)

	// Попытка опубликовать должна вернуть ошибку
	msg := queue.Message{
		Topic: s.topic,
		Body:  "test",
	}

	err = pub.Publish(ctx, msg)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "closed")
}

func (s *KafkaSuite) TestPublisher_DefaultBalancer() {
	dialer := s.createDialer()

	// Создаем publisher без указания балансировщика
	pub := kafka.NewPublisher(dialer, kafka.PublisherConfig{})

	assert.NotNil(s.T(), pub)
	require.NoError(s.T(), pub.Close())
	require.NoError(s.T(), dialer.Close())
}

func (s *KafkaSuite) TestPublisher_WithLeastBytesBalancer() {
	dialer := s.createDialer()

	pub := kafka.NewPublisher(dialer, kafka.PublisherConfig{
		Balancer: &kafkago.LeastBytes{},
	})

	assert.NotNil(s.T(), pub)
	require.NoError(s.T(), pub.Close())
	require.NoError(s.T(), dialer.Close())
}
