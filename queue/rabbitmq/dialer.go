package rabbitmq

import (
	"log/slog"
	"sync"
	"time"

	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"
)

var ErrConnectionClosed = errors.New("connection is closed manually")

type Dialer struct {
	uri     string
	conn    *amqp.Connection
	options *DialerOptions
	mx      sync.Mutex
}

// RetryPolicy of dialer reconnection.
type RetryPolicy interface {
	TryNum(i int) (duration time.Duration, stop bool)
}

// DialerOptions set dialer params.
type DialerOptions struct {
	RetryPolicy RetryPolicy
	Logger      *slog.Logger
}

func NewDefaultDialer(uri string) *Dialer {
	return NewDialer(uri, nil)
}

func NewDialer(uri string, options *DialerOptions) *Dialer {
	if options == nil {
		options = new(DialerOptions)
	}
	if options.Logger == nil {
		options.Logger = slog.Default()
	}
	options.Logger = options.Logger.WithGroup("rabbitmq")

	if options.RetryPolicy == nil {
		options.RetryPolicy = NewDefaultMaxInterval()
	}

	return &Dialer{
		uri:     uri,
		options: options,
	}
}

func (d *Dialer) Connect() (err error) {
	d.options.Logger.Debug("Dialing...")

	d.mx.Lock()
	defer d.mx.Unlock()

	conn, err := amqp.DialConfig(d.uri, amqp.Config{})
	if err != nil {
		return errors.Wrap(err, "failed to dial")
	}

	ch := conn.NotifyClose(make(chan *amqp.Error, 1))
	d.conn = conn
	d.options.Logger.Debug("Connection is stable")
	go d.handleReconnect(ch)
	return nil
}

func (d *Dialer) Channel() (*amqp.Channel, error) {
	d.mx.Lock()
	defer d.mx.Unlock()

	if d.conn == nil {
		return nil, ErrConnectionClosed
	}

	channel, err := d.conn.Channel()
	if err != nil {
		return nil, errors.Wrap(err, "failed to open channel")
	}
	return channel, nil
}

func (d *Dialer) Close() error {
	d.mx.Lock()
	defer d.mx.Unlock()

	if d.conn == nil {
		return nil
	}
	if err := d.conn.Close(); err != nil {
		return errors.Wrap(err, "failed to close RabbitMQ connection")
	}
	d.conn = nil
	return nil
}

// handleReconnect listens AMQP connection failures from go-chan and attempts to reconnect
func (d *Dialer) handleReconnect(ch chan *amqp.Error) {
	err, ok := <-ch
	if !ok {
		d.options.Logger.Debug("Shutdown")
		return
	}

	d.options.Logger.With("error", err.Error()).Warn("Disconnected")

	for i := 0; ; i++ {
		err := d.Connect()
		if err == nil {
			return
		}

		sleepDuration, stop := d.options.RetryPolicy.TryNum(i)
		if stop {
			d.options.Logger.With("error", err.Error()).Error("Cannot to connect to rabbitmq. Time is out")
			return
		} else {
			d.options.Logger.With("error", err.Error()).Error("Failed to connect")
		}

		time.Sleep(sleepDuration)
	}
}
