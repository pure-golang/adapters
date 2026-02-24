package kafka_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/suite"
	kafkatestcontainers "github.com/testcontainers/testcontainers-go/modules/kafka"

	"github.com/pure-golang/adapters/queue"
	"github.com/pure-golang/adapters/queue/kafka"
)

type KafkaSuite struct {
	suite.Suite
	brokers        []string
	topic          string
	conn           *kafkago.Conn
	kafkaContainer *kafkatestcontainers.KafkaContainer
}

func TestKafkaSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	suite.Run(t, new(KafkaSuite))
}

func (s *KafkaSuite) SetupSuite() {
	ctx := context.Background()

	// Запускаем Kafka контейнер через testcontainers
	kafkaContainer, err := kafkatestcontainers.Run(ctx, "confluentinc/cp-kafka:7.6.0",
		kafkatestcontainers.WithClusterID("test-cluster-"+uuid.NewString()),
	)
	s.Require().NoError(err, "failed to start Kafka container")

	s.kafkaContainer = kafkaContainer

	// Получаем адрес брокеров
	brokers, err := kafkaContainer.Brokers(ctx)
	s.Require().NoError(err, "failed to get Kafka brokers")

	s.brokers = brokers
	s.topic = "test-topic-" + uuid.NewString()

	// Ждем готовности Kafka
	var errConnect error
	for range 30 {
		s.conn, errConnect = kafkago.Dial("tcp", s.brokers[0])
		if errConnect == nil {
			break
		}
		s.T().Logf("Waiting for Kafka... (%v)", errConnect)
		time.Sleep(1 * time.Second)
	}
	s.Require().NoError(errConnect, "Kafka connection timeout after 30 seconds")

	// Создаем тему для тестов
	err = s.conn.CreateTopics(kafkago.TopicConfig{
		Topic:             s.topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil {
		s.T().Logf("Warning: failed to create topic (may already exist): %v", err)
	}
}

func (s *KafkaSuite) TearDownSuite() {
	if s.conn != nil {
		if err := s.conn.Close(); err != nil {
			s.T().Logf("Failed to close connection: %v", err)
		}
	}

	if s.kafkaContainer != nil {
		ctx := context.Background()
		if err := s.kafkaContainer.Terminate(ctx); err != nil {
			s.T().Logf("Failed to terminate Kafka container: %v", err)
		}
	}
}

// createDialer создает dialer для тестов
func (s *KafkaSuite) createDialer() *kafka.Dialer {
	cfg := kafka.Config{
		Brokers: s.brokers,
	}
	return kafka.NewDialer(cfg)
}

// createPublisher создает publisher для тестов
func (s *KafkaSuite) createPublisher(encoder queue.Encoder) *kafka.Publisher {
	dialer := s.createDialer()
	s.T().Cleanup(func() {
		dialer.Close()
	})

	pub := kafka.NewPublisher(dialer, kafka.PublisherConfig{
		Encoder: encoder,
	})
	s.T().Cleanup(func() {
		pub.Close()
	})

	return pub
}
