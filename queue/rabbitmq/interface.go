package rabbitmq

import (
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

// channelProvider открывает новый AMQP-канал по требованию.
//
//go:generate mockery --name=channelProvider --exported
type channelProvider interface {
	Channel() (*amqp.Channel, error)
	Logger() *slog.Logger
}
