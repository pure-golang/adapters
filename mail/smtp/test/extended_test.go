package smtp_test

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/pure-golang/adapters/mail"
	smtpadapter "github.com/pure-golang/adapters/mail/smtp"
)

func startSMTPContainer(t *testing.T) testcontainers.Container {
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "mailhog/mailhog:latest",
		ExposedPorts: []string{"1025/tcp", "8025/tcp"},
		WaitingFor:   wait.ForListeningPort("1025/tcp"),
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

func waitForSMTP(t *testing.T, host string, port int) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	deadline := time.Now().Add(30 * time.Second)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			conn.Close()
			time.Sleep(500 * time.Millisecond)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("SMTP server at %s did not become ready", addr)
}

func TestSender_Extended_NoTLS(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := smtpadapter.Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := smtpadapter.NewSender(cfg)
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

func TestSender_Extended_WithDefaultFrom(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := smtpadapter.Config{
		Host: host,
		Port: port,
		From: "default@example.com",
		TLS:  false,
	}

	sender := smtpadapter.NewSender(cfg)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Default From Test",
		Body:    "Testing default From address.",
	}

	err := sender.Send(ctx, email)
	require.NoError(t, err)
}

func TestSender_Extended_ExplicitFromOverridesDefault(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := smtpadapter.Config{
		Host: host,
		Port: port,
		From: "default@example.com",
		TLS:  false,
	}

	sender := smtpadapter.NewSender(cfg)
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

func TestSender_Extended_MultipleEmails(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := smtpadapter.Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := smtpadapter.NewSender(cfg)
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

func TestSender_Extended_MultipleRecipients(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := smtpadapter.Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := smtpadapter.NewSender(cfg)
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

func TestSender_Extended_OnlyCcRecipients(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := smtpadapter.Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := smtpadapter.NewSender(cfg)
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

func TestSender_Extended_OnlyBccRecipients(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := smtpadapter.Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := smtpadapter.NewSender(cfg)
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

func TestSender_Extended_HTMLWithMultipart(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := smtpadapter.Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := smtpadapter.NewSender(cfg)
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

func TestSender_Extended_MultipleCustomHeaders(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := smtpadapter.Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := smtpadapter.NewSender(cfg)
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

func TestSender_Extended_EmptySubject(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := smtpadapter.Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := smtpadapter.NewSender(cfg)
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

func TestSender_Extended_LongSubject(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := smtpadapter.Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := smtpadapter.NewSender(cfg)
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

func TestSender_Extended_SpecialCharactersInBody(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := smtpadapter.Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := smtpadapter.NewSender(cfg)
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

func TestSender_Extended_UnicodeContent(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := smtpadapter.Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := smtpadapter.NewSender(cfg)
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

func TestSender_Extended_SenderConcurrency(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := smtpadapter.Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := smtpadapter.NewSender(cfg)
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

	for i := 0; i < numGoroutines; i++ {
		err := <-errChan
		assert.NoError(t, err)
	}
}

func TestSender_Extended_ConnectionRefused(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg := smtpadapter.Config{
		Host: "localhost",
		Port: 9999,
		TLS:  false,
	}

	sender := smtpadapter.NewSender(cfg)
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

func TestSender_Extended_InvalidHost(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg := smtpadapter.Config{
		Host: "invalid-host-that-does-not-exist.local",
		Port: 2525,
		TLS:  false,
	}

	sender := smtpadapter.NewSender(cfg)
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

func TestSender_Extended_ContextTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	cfg := smtpadapter.Config{
		Host: "192.0.2.1", // TEST-NET-1 - unreachable
		Port: 2525,
		TLS:  false,
	}

	sender := smtpadapter.NewSender(cfg)
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

func TestSender_Extended_SMTPDirect(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	addr := net.JoinHostPort(host, strconv.Itoa(port))

	client, err := smtp.Dial(addr)
	require.NoError(t, err)
	defer client.Close()

	err = client.Noop()
	require.NoError(t, err)

	err = client.Mail("test@example.com")
	require.NoError(t, err)

	err = client.Rcpt("recipient@example.com")
	require.NoError(t, err)

	writer, err := client.Data()
	require.NoError(t, err)

	_, err = writer.Write([]byte("Subject: Test\r\n\r\nTest body\r\n"))
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)
}

func TestSender_Extended_CloseIdempotent(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := smtpadapter.Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := smtpadapter.NewSender(cfg)

	err1 := sender.Close()
	err2 := sender.Close()
	err3 := sender.Close()

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
}

func TestSender_Extended_BodyOnlyNoHTML(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := smtpadapter.Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := smtpadapter.NewSender(cfg)
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
}

func TestSender_Extended_HTMLNoBody(t *testing.T) {
	container := startSMTPContainer(t)
	host, port := getSMTPHostPort(t, container)
	waitForSMTP(t, host, port)

	cfg := smtpadapter.Config{
		Host: host,
		Port: port,
		TLS:  false,
	}

	sender := smtpadapter.NewSender(cfg)
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
}
