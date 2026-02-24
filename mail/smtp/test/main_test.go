package smtp_test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/pure-golang/adapters/mail"
	"github.com/pure-golang/adapters/mail/smtp"
)

var testSender *smtp.Sender

func TestMain(m *testing.M) {
	flag.Parse()

	if testing.Short() {
		fmt.Println("Skipping integration tests in short mode")
		os.Exit(0)
	}

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "mailhog/mailhog:latest",
		ExposedPorts: []string{"1025/tcp"},
		WaitingFor:   wait.ForListeningPort("1025/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		panic(fmt.Sprintf("could not start mailhog container: %s", err))
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		panic(fmt.Sprintf("could not get container host: %s", err))
	}

	port, err := container.MappedPort(ctx, "1025")
	if err != nil {
		_ = container.Terminate(ctx)
		panic(fmt.Sprintf("could not get container port: %s", err))
	}

	portNum, err := strconv.Atoi(port.Port())
	if err != nil {
		_ = container.Terminate(ctx)
		panic(fmt.Sprintf("could not parse port number: %s", err))
	}

	cfg := smtp.Config{
		Host: host,
		Port: portNum,
		TLS:  false,
	}

	testSender = smtp.NewSender(cfg)

	code := m.Run()

	if testSender != nil {
		testSender.Close()
	}

	if err := container.Terminate(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to terminate container: %v\n", err)
	}

	os.Exit(code)
}

func TestSender_Integration_Send_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	email := mail.Email{
		From:    mail.Address{Name: "Test Sender", Address: "test@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Integration Test Email",
		Body:    "This is a test email from integration tests.",
	}

	err := testSender.Send(ctx, email)
	require.NoError(t, err)
}

func TestSender_Integration_Send_WithName(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	email := mail.Email{
		From: mail.Address{Name: "John Doe", Address: "john@example.com"},
		To: []mail.Address{
			{Name: "Jane Smith", Address: "jane@example.com"},
			{Name: "Bob Johnson", Address: "bob@example.com"},
		},
		Subject: "Test Email with Names",
		Body:    "Testing email with recipient names.",
	}

	err := testSender.Send(ctx, email)
	require.NoError(t, err)
}

func TestSender_Integration_Send_WithHTML(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	email := mail.Email{
		From:    mail.Address{Address: "test@example.com"},
		To:      []mail.Address{{Address: "html@example.com"}},
		Subject: "HTML Email Test",
		Body:    "Plain text version",
		HTML:    "<html><body><h1>HTML Version</h1><p>This is the HTML version.</p></body></html>",
	}

	err := testSender.Send(ctx, email)
	require.NoError(t, err)
}

func TestSender_Integration_Send_WithCcAndBcc(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	email := mail.Email{
		From:    mail.Address{Address: "test@example.com"},
		To:      []mail.Address{{Address: "to@example.com"}},
		Cc:      []mail.Address{{Address: "cc@example.com"}},
		Bcc:     []mail.Address{{Address: "bcc@example.com"}},
		Subject: "Test with CC and BCC",
		Body:    "Testing CC and BCC recipients.",
	}

	err := testSender.Send(ctx, email)
	require.NoError(t, err)
}

func TestSender_Integration_Send_WithCustomHeaders(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	email := mail.Email{
		From:    mail.Address{Address: "test@example.com"},
		To:      []mail.Address{{Address: "headers@example.com"}},
		Subject: "Test with Custom Headers",
		Body:    "Testing custom headers.",
		Headers: map[string]string{
			"X-Priority": "1",
			"X-Custom":   "custom-value",
		},
	}

	err := testSender.Send(ctx, email)
	require.NoError(t, err)
}

func TestSender_Integration_Send_MultipleEmails(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	emails := []mail.Email{
		{
			From:    mail.Address{Address: "test1@example.com"},
			To:      []mail.Address{{Address: "recipient1@example.com"}},
			Subject: "Multiple Test 1",
			Body:    "First email in batch.",
		},
		{
			From:    mail.Address{Address: "test2@example.com"},
			To:      []mail.Address{{Address: "recipient2@example.com"}},
			Subject: "Multiple Test 2",
			Body:    "Second email in batch.",
		},
	}

	err := testSender.Send(ctx, emails...)
	require.NoError(t, err)
}

func TestSender_Integration_Send_WithSpecialCharacters(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	email := mail.Email{
		From:    mail.Address{Name: "Тестовый Отправитель", Address: "test@example.com"},
		To:      []mail.Address{{Name: "Получатель Тест", Address: "recipient@example.com"}},
		Subject: "Тема с кириллицей и символами: <>{}[]",
		Body:    "Email body with special characters: <>&\"'",
	}

	err := testSender.Send(ctx, email)
	require.NoError(t, err)
}

func TestSender_Integration_Send_LongSubject(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	ctx := context.Background()

	longSubject := "This is a very long subject line that might exceed typical email header length limits " +
		"and should still be handled properly by the SMTP sender implementation " +
		"without causing any issues with the mail server"

	email := mail.Email{
		From:    mail.Address{Address: "test@example.com"},
		To:      []mail.Address{{Address: "long@example.com"}},
		Subject: longSubject,
		Body:    "Testing long subject line.",
	}

	err := testSender.Send(ctx, email)
	require.NoError(t, err)
}
