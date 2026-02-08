package kafka

// Config содержит параметры подключения к Kafka
type Config struct {
	Brokers []string `envconfig:"KAFKA_BROKERS" required:"true"` // список брокеров Kafka (например: localhost:9092)
	GroupID string   `envconfig:"KAFKA_GROUP_ID"`                // идентификатор группы потребителей
}

// DialerConfig содержит параметры для Dialer
type DialerConfig struct {
	Timeout     int `envconfig:"KAFKA_DIALER_TIMEOUT" default:"10"` // таймаут подключения в секундах
	MaxAttempts int `envconfig:"KAFKA_MAX_ATTEMPTS" default:"3"`    // максимальное количество попыток подключения
}
