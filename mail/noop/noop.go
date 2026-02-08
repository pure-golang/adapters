package noop

import (
	"context"

	"github.com/pure-golang/adapters/mail"
)

var _ mail.Sender = (*Sender)(nil)

// Sender is a no-op mail sender for testing.
type Sender struct {
	closed bool
}

// NewSender creates a new no-op Sender.
func NewSender() *Sender {
	return &Sender{
		closed: false,
	}
}

// Send silently discards emails.
func (n *Sender) Send(ctx context.Context, emails ...mail.Email) error {
	for _, email := range emails {
		_ = email // Discard
	}
	return nil
}

// Close is a no-op.
func (n *Sender) Close() error {
	n.closed = true
	return nil
}
