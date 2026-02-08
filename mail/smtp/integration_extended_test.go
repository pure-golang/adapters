//go:build integration
// +build integration

package smtp

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pure-golang/adapters/mail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// skipShort skips the test in short mode
func skipShortExtended(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping extended integration test in short mode")
	}
}

// startSMTPContainer starts a MailHog container for testing
func startSMTPContainer(t *testing.T) testcontainers.Container {
	skipShortExtended(t)

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "mailhog/mailhog:latest",
		ExposedPorts: []string{"1025/tcp", "8025/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForLog("Starting SMTP"),
			wait.ForListeningPort("1025/tcp"),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "failed to start mailhog container")

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	return container
}

// getSMTPHostPort returns the host and port for the SMTP server
func getSMTPHostPort(t *testing.T, container testcontainers.Container) (string, int) {
	ctx := context.Background()
	host, err := container.Host(ctx)
	require.NoError(t, err, "failed to get container host")

	port, err := container.MappedPort(ctx, "1025")
	require.NoError(t, err, "failed to get container port")

	portNum, err := strconv.Atoi(port.Port())
	require.NoError(t, err, "failed to parse port number")

	return host, portNum
}

// waitForSMTP waits for the SMTP server to be ready
func waitForSMTP(t *testing.T, host string, port int) {
	addr := fmt.Sprintf("%s:%d", host, port)
	deadline := time.Now().Add(30 * time.Second)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			conn.Close()
			// Give the server a moment to be fully ready
			time.Sleep(500 * time.Millisecond)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("SMTP server at %s did not become ready", addr)
}

// TestSender_Extended_NoTLS tests sending emails without TLS
func TestSender_Extended_NoTLS(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Name: "Test Sender", Address: "test@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "No TLS Test",
		Body:    "This is a test email without TLS.",
	}

	err := sender.Send(ctx, email)
	require.NoError(t, err)
}

// TestSender_Extended_WithDefaultFrom tests using default From address
func TestSender_Extended_WithDefaultFrom(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := Config{
		Host: host,
		Port: port,
		From: "default@example.com",
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		// No From specified - should use default
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Default From Test",
		Body:    "Testing default From address.",
	}

	err := sender.Send(ctx, email)
	require.NoError(t, err)
}

// TestSender_Extended_ExplicitFromOverridesDefault tests that explicit From overrides default
func TestSender_Extended_ExplicitFromOverridesDefault(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := Config{
		Host: host,
		Port: port,
		From: "default@example.com",
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "explicit@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Explicit From Test",
		Body:    "Testing explicit From address.",
	}

	err := sender.Send(ctx, email)
	require.NoError(t, err)
}

// TestSender_Extended_MultipleEmails tests sending multiple emails in one call
func TestSender_Extended_MultipleEmails(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	emails := []mail.Email{
		{
			From:    mail.Address{Address: "sender1@example.com"},
			To:      []mail.Address{{Address: "recipient1@example.com"}},
			Subject: "Multiple Test 1",
			Body:    "First email in batch.",
		},
		{
			From:    mail.Address{Address: "sender2@example.com"},
			To:      []mail.Address{{Address: "recipient2@example.com"}},
			Subject: "Multiple Test 2",
			Body:    "Second email in batch.",
		},
		{
			From:    mail.Address{Address: "sender3@example.com"},
			To:      []mail.Address{{Address: "recipient3@example.com"}},
			Subject: "Multiple Test 3",
			Body:    "Third email in batch.",
		},
	}

	err := sender.Send(ctx, emails...)
	require.NoError(t, err)
}

// TestSender_Extended_MultipleRecipients tests sending to multiple recipients
func TestSender_Extended_MultipleRecipients(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From: mail.Address{Address: "sender@example.com"},
		To: []mail.Address{
			{Address: "to1@example.com"},
			{Address: "to2@example.com"},
			{Address: "to3@example.com"},
		},
		Cc: []mail.Address{
			{Address: "cc1@example.com"},
			{Address: "cc2@example.com"},
		},
		Bcc: []mail.Address{
			{Address: "bcc1@example.com"},
			{Address: "bcc2@example.com"},
		},
		Subject: "Multiple Recipients Test",
		Body:    "Testing multiple recipients.",
	}

	err := sender.Send(ctx, email)
	require.NoError(t, err)
}

// TestSender_Extended_OnlyCcRecipients tests sending with only CC recipients
func TestSender_Extended_OnlyCcRecipients(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		Cc:      []mail.Address{{Address: "cc@example.com"}},
		Subject: "Only CC Test",
		Body:    "Testing only CC recipients.",
	}

	err := sender.Send(ctx, email)
	require.NoError(t, err)
}

// TestSender_Extended_OnlyBccRecipients tests sending with only BCC recipients
func TestSender_Extended_OnlyBccRecipients(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		Bcc:     []mail.Address{{Address: "bcc@example.com"}},
		Subject: "Only BCC Test",
		Body:    "Testing only BCC recipients.",
	}

	err := sender.Send(ctx, email)
	require.NoError(t, err)
}

// TestSender_Extended_HTMLWithMultipart tests sending HTML email with multipart
func TestSender_Extended_HTMLWithMultipart(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "test@example.com"},
		To:      []mail.Address{{Address: "html@example.com"}},
		Subject: "HTML Multipart Test",
		Body:    "Plain text version for email clients that don't support HTML.",
		HTML:    `<html><body><h1>HTML Version</h1><p>This is the <strong>HTML</strong> version.</p></body></html>`,
	}

	err := sender.Send(ctx, email)
	require.NoError(t, err)
}

// TestSender_Extended_MultipleCustomHeaders tests sending with multiple custom headers
func TestSender_Extended_MultipleCustomHeaders(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "test@example.com"},
		To:      []mail.Address{{Address: "headers@example.com"}},
		Subject: "Multiple Custom Headers Test",
		Body:    "Testing multiple custom headers.",
		Headers: map[string]string{
			"X-Priority":               "1 (Highest)",
			"X-MSMail-Priority":        "High",
			"X-Mailer":                 "TestMailer/1.0",
			"X-Auto-Response-Suppress": "All",
			"References":               "<msg1@example.com> <msg2@example.com>",
		},
	}

	err := sender.Send(ctx, email)
	require.NoError(t, err)
}

// TestSender_Extended_EmptySubject tests sending email with empty subject
func TestSender_Extended_EmptySubject(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "test@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "",
		Body:    "Testing empty subject.",
	}

	err := sender.Send(ctx, email)
	require.NoError(t, err)
}

// TestSender_Extended_LongSubject tests sending email with very long subject
func TestSender_Extended_LongSubject(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	longSubject := strings.Repeat("This is a very long subject line. ", 20)

	email := mail.Email{
		From:    mail.Address{Address: "test@example.com"},
		To:      []mail.Address{{Address: "long@example.com"}},
		Subject: longSubject,
		Body:    "Testing long subject line.",
	}

	err := sender.Send(ctx, email)
	require.NoError(t, err)
}

// TestSender_Extended_SpecialCharactersInBody tests sending email with special characters
func TestSender_Extended_SpecialCharactersInBody(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "test@example.com"},
		To:      []mail.Address{{Address: "special@example.com"}},
		Subject: "Special Characters Test",
		Body:    "Testing special characters: <>&\"'`\n\tNewlines and tabs.\nMultiple\nlines.",
	}

	err := sender.Send(ctx, email)
	require.NoError(t, err)
}

// TestSender_Extended_UnicodeContent tests sending email with Unicode content
func TestSender_Extended_UnicodeContent(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Name: "Отправитель", Address: "test@example.com"},
		To:      []mail.Address{{Name: "Получатель", Address: "recipient@example.com"}},
		Subject: "Тест с Unicode: 中文, 日本語, 한국어",
		Body:    "Тело письма с кириллицей и другими символами: العربية, עברית",
	}

	err := sender.Send(ctx, email)
	require.NoError(t, err)
}

// TestSender_Extended_SenderConcurrency tests concurrent sends
func TestSender_Extended_SenderConcurrency(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	const numGoroutines = 10
	const emailsPerGoroutine = 5

	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			for j := 0; j < emailsPerGoroutine; j++ {
				email := mail.Email{
					From:    mail.Address{Address: fmt.Sprintf("sender%d@example.com", idx)},
					To:      []mail.Address{{Address: fmt.Sprintf("recipient%d@example.com", idx*emailsPerGoroutine+j)}},
					Subject: fmt.Sprintf("Concurrent Test %d-%d", idx, j),
					Body:    fmt.Sprintf("Concurrent email %d-%d", idx, j),
				}
				if err := sender.Send(ctx, email); err != nil {
					errChan <- err
					return
				}
			}
			errChan <- nil
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		err := <-errChan
		assert.NoError(t, err)
	}
}

// TestSender_Extended_VerifyMessageFormat verifies the message format using buildMessage
func TestSender_Extended_VerifyMessageFormat(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	email := mail.Email{
		From:    mail.Address{Name: "Test Sender", Address: "sender@example.com"},
		To:      []mail.Address{{Name: "Recipient", Address: "recipient@example.com"}},
		Cc:      []mail.Address{{Address: "cc@example.com"}},
		Subject: "Format Verification Test",
		Body:    "Plain text body",
		HTML:    "<html><body>HTML body</body></html>",
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
		},
	}

	// Build message directly to verify format
	msg := sender.buildMessage(email)
	msgStr := string(msg)

	// Verify headers
	assert.Contains(t, msgStr, "From:")
	assert.Contains(t, msgStr, "To:")
	assert.Contains(t, msgStr, "Cc:")
	assert.Contains(t, msgStr, "Subject:")
	assert.Contains(t, msgStr, "MIME-Version: 1.0")
	assert.Contains(t, msgStr, "Date:")
	assert.Contains(t, msgStr, "X-Custom-Header: custom-value")

	// Verify multipart/alternative for HTML
	assert.Contains(t, msgStr, "multipart/alternative")
	assert.Contains(t, msgStr, "boundary_")

	// Verify both plain text and HTML parts
	assert.Contains(t, msgStr, "Plain text body")
	assert.Contains(t, msgStr, "HTML body")
}

// TestSender_Extended_ConnectionRefused tests connection refused error
func TestSender_Extended_ConnectionRefused(t *testing.T) {
	cfg := Config{
		Host: "localhost",
		Port: 9999, // Non-existent server
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "test@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Connection Refused Test",
		Body:    "Testing connection refused.",
	}

	err := sender.Send(ctx, email)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send email")
}

// TestSender_Extended_InvalidHost tests invalid host error
func TestSender_Extended_InvalidHost(t *testing.T) {
	cfg := Config{
		Host: "invalid-host-that-does-not-exist.local",
		Port: 2525,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	email := mail.Email{
		From:    mail.Address{Address: "test@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Invalid Host Test",
		Body:    "Testing invalid host.",
	}

	err := sender.Send(ctx, email)
	assert.Error(t, err)
}

// TestSender_Extended_ContextTimeout tests context timeout
func TestSender_Extended_ContextTimeout(t *testing.T) {
	cfg := Config{
		Host: "192.0.2.1", // TEST-NET-1 - should be unreachable
		Port: 2525,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	email := mail.Email{
		From:    mail.Address{Address: "test@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Context Timeout Test",
		Body:    "Testing context timeout.",
	}

	err := sender.Send(ctx, email)
	assert.Error(t, err)
}

// TestSender_Extended_SMTPDirect verifies SMTP operations work correctly
func TestSender_Extended_SMTPDirect(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	addr := fmt.Sprintf("%s:%d", host, port)

	// Test basic SMTP connection using net/smtp directly
	client, err := smtp.Dial(addr)
	require.NoError(t, err)
	defer client.Close()

	// Test NOOP
	err = client.Noop()
	require.NoError(t, err)

	// Test MAIL
	err = client.Mail("test@example.com")
	require.NoError(t, err)

	// Test RCPT
	err = client.Rcpt("recipient@example.com")
	require.NoError(t, err)

	// Test DATA
	writer, err := client.Data()
	require.NoError(t, err)

	_, err = writer.Write([]byte("Subject: Test\r\n\r\nTest body\r\n"))
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)
}

// TestSender_Extended_CloseIdempotent tests that Close can be called multiple times
func TestSender_Extended_CloseIdempotent(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)

	// Close multiple times
	err1 := sender.Close()
	err2 := sender.Close()
	err3 := sender.Close()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
}

// TestSender_Extended_BuildMessageVerifyBoundary tests that boundary is unique
func TestSender_Extended_BuildMessageVerifyBoundary(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	// Build multiple messages and verify boundaries are different
	email := mail.Email{
		From:    mail.Address{Address: "test@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Boundary Test",
		Body:    "Plain",
		HTML:    "<p>HTML</p>",
	}

	// Add small delays to ensure different timestamps
	msg1 := sender.buildMessage(email)
	time.Sleep(time.Millisecond)
	msg2 := sender.buildMessage(email)
	time.Sleep(time.Millisecond)
	msg3 := sender.buildMessage(email)

	msg1Str := string(msg1)
	msg2Str := string(msg2)
	msg3Str := string(msg3)

	// Extract boundaries
	getBoundary := func(msg string) string {
		parts := strings.Split(msg, "boundary_")
		if len(parts) < 2 {
			return ""
		}
		boundaryPart := strings.Split(parts[1], "\r\n")[0]
		return "boundary_" + boundaryPart
	}

	b1 := getBoundary(msg1Str)
	b2 := getBoundary(msg2Str)
	b3 := getBoundary(msg3Str)

	// Boundaries should be different (due to time-based generation)
	assert.NotEqual(t, b1, b2, "boundaries should be unique")
	assert.NotEqual(t, b2, b3, "boundaries should be unique")
}

// TestSender_Extended_BodyOnlyNoHTML tests sending plain text email without HTML
func TestSender_Extended_BodyOnlyNoHTML(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "test@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Plain Text Only Test",
		Body:    "This is a plain text email without HTML.",
	}

	err := sender.Send(ctx, email)
	require.NoError(t, err)

	// Verify the message doesn't have multipart
	msg := sender.buildMessage(email)
	msgStr := string(msg)
	assert.NotContains(t, msgStr, "multipart/alternative")
	assert.Contains(t, msgStr, "Content-Type: text/plain")
}

// TestSender_Extended_HTMLNoBody tests sending HTML email without plain text body
func TestSender_Extended_HTMLNoBody(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "test@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "HTML Only Test",
		HTML:    "<html><body><h1>HTML Only</h1></body></html>",
	}

	err := sender.Send(ctx, email)
	require.NoError(t, err)

	// Verify the message has multipart with both parts
	msg := sender.buildMessage(email)
	msgStr := string(msg)
	assert.Contains(t, msgStr, "multipart/alternative")
}
