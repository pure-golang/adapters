package noop

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pure-golang/adapters/mail"
)

func TestSender_Send(t *testing.T) {
	sender := NewSender()

	ctx := context.Background()
	emails := []mail.Email{
		{
			From:    mail.Address{Address: "test@example.com"},
			To:      []mail.Address{{Address: "to@example.com"}},
			Subject: "Test",
			Body:    "Test body",
		},
	}

	err := sender.Send(ctx, emails...)
	assert.NoError(t, err)
}

func TestSender_Send_EmptyList(t *testing.T) {
	sender := NewSender()

	ctx := context.Background()
	err := sender.Send(ctx)
	assert.NoError(t, err)
}

func TestSender_Close(t *testing.T) {
	sender := NewSender()

	err := sender.Close()
	assert.NoError(t, err)

	err = sender.Close()
	assert.NoError(t, err)
}
