//go:build integration
// +build integration

package smtp

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/pure-golang/adapters/mail"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
)

var testSender *Sender

func TestMain(m *testing.M) {
	flag.Parse()

	if testing.Short() {
		fmt.Println("Skipping integration tests in short mode")
		os.Exit(0)
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		panic(fmt.Sprintf("Could not connect to docker: %s", err))
	}

	// Use mailhog/mailhog with explicit port binding
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "mailhog/mailhog",
		Tag:        "latest",
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		config.PortBindings = map[docker.Port][]docker.PortBinding{
			"1025/tcp": {{HostIP: "", HostPort: "1025"}},
		}
	})
	if err != nil {
		panic(fmt.Sprintf("Could not start resource: %s", err))
	}

	defer pool.Purge(resource)

	port, err := strconv.Atoi(resource.GetPort("1025/tcp"))
	if err != nil {
		panic(fmt.Sprintf("Could not parse port: %s", err))
	}

	cfg := Config{
		Host: "localhost",
		Port: port,
		TLS:  false,
	}

	// Wait for SMTP to be ready
	if err := pool.Retry(func() error {
		addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err != nil {
			return err
		}
		conn.Close()
		return nil
	}); err != nil {
		panic(fmt.Sprintf("Could not connect to SMTP: %s", err))
	}

	// Wait a bit for the server to be fully ready
	time.Sleep(1 * time.Second)

	testSender = NewSender(cfg, nil)

	code := m.Run()

	if testSender != nil {
		testSender.Close()
	}

	os.Exit(code)
}

func skipShort(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
}

func TestSender_Integration_Send_Success(t *testing.T) {
	skipShort(t)
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
	skipShort(t)
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
	skipShort(t)
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
	skipShort(t)
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
	skipShort(t)
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
	skipShort(t)
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
	skipShort(t)
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
	skipShort(t)
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
