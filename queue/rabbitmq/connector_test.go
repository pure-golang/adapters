package rabbitmq

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pure-golang/adapters/queue/rabbitmq/mocks"
)

func TestConnectWithRetry(t *testing.T) {
	t.Parallel()

	// Arrange
	d := mocks.NewDialer(t)
	d.EXPECT().Connect().Return(nil)
	connector := NewConnector(d)

	// Act
	err := connector.ConnectWithRetry(context.Background())

	// Assert
	require.NoError(t, err)
}

func TestConnectWithRetry_context_cancelled(t *testing.T) {
	t.Parallel()

	// Arrange
	d := mocks.NewDialer(t)
	d.EXPECT().Connect().Return(assert.AnError)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	connector := NewConnector(d)

	// Act
	err := connector.ConnectWithRetry(ctx)

	// Assert
	assert.ErrorIs(t, err, context.Canceled)
}

func TestConnectWithRetry_retry_succeeds(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		// Arrange
		d := mocks.NewDialer(t)
		d.EXPECT().Connect().Return(assert.AnError).Once()
		d.EXPECT().Connect().Return(nil).Once()
		connector := NewConnector(d)

		// Act
		err := connector.ConnectWithRetry(context.Background())

		// Assert
		require.NoError(t, err)
	})
}

func TestWaitForExchange_context_cancelled(t *testing.T) {
	t.Parallel()

	// Arrange
	d := mocks.NewDialer(t)
	d.EXPECT().Channel().Return(nil, assert.AnError)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	connector := NewConnector(d)

	// Act
	err := connector.WaitForExchange(ctx, "calls")

	// Assert
	assert.ErrorIs(t, err, context.Canceled)
}

func TestWaitForQueue_context_cancelled(t *testing.T) {
	t.Parallel()

	// Arrange
	d := mocks.NewDialer(t)
	d.EXPECT().Channel().Return(nil, assert.AnError)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	connector := NewConnector(d)

	// Act
	err := connector.WaitForQueue(ctx, "calls.events")

	// Assert
	assert.ErrorIs(t, err, context.Canceled)
}
