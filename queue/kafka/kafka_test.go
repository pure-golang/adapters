package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/suite"

	kafkatestcontainers "github.com/testcontainers/testcontainers-go/modules/kafka"
)

type KafkaSuite struct {
	suite.Suite
	brokers        []string
	topic          string
	conn           *kafkago.Conn
	kafkaContainer *kafkatestcontainers.KafkaContainer
}

func TestKafkaSuite(t *testing.T) {
	suite.Run(t, new(KafkaSuite))
}

func (s *KafkaSuite) SetupSuite() {
	if testing.Short() {
		s.T().Skip("integration test is skipped")
	}

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
	for i := 0; i < 30; i++ {
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

func (s *KafkaSuite) TestHeadersCarrier() {
	headers := make(map[string]string)
	headers["traceparent"] = "00-12345678901234567890123456789012-1234567890123456-01"
	headers["custom-header"] = "test-value"

	carrier := headersCarrier(headers)

	s.Require().Equal("00-12345678901234567890123456789012-1234567890123456-01", carrier.Get("traceparent"))
	s.Require().Equal("test-value", carrier.Get("custom-header"))
	s.Require().Equal("", carrier.Get("non-existent"))
}

func (s *KafkaSuite) TestHeadersCarrier_Empty() {
	headers := make(map[string]string)
	carrier := headersCarrier(headers)

	s.Require().Equal("", carrier.Get("traceparent"))
	s.Require().Equal(0, len(carrier.Keys()))
}
