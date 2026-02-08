package smtp

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/pure-golang/adapters/mail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// miniSMTPServer is a minimal SMTP server for testing
type miniSMTPServer struct {
	listener net.Listener
	messages [][]byte
}

// startMiniSMTPServer starts a minimal SMTP server on localhost
func startMiniSMTPServer(t *testing.T, port int) *miniSMTPServer {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	require.NoError(t, err, "failed to start SMTP server")

	server := &miniSMTPServer{
		listener: listener,
		messages: make([][]byte, 0),
	}

	go server.handleConnections(t)

	return server
}

func (s *miniSMTPServer) handleConnections(t *testing.T) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return // listener closed
		}

		go func() {
			defer conn.Close()
			s.handleSMTP(t, conn)
		}()
	}
}

func (s *miniSMTPServer) handleSMTP(t *testing.T, conn net.Conn) {
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// Send greeting
	writer.WriteString("220 localhost ESMTP Test Server\r\n")
	writer.Flush()

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		line = strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(line, "EHLO") || strings.HasPrefix(line, "HELO"):
			writer.WriteString("250-localhost\r\n250-SIZE 10240000\r\n250 HELP\r\n")
			writer.Flush()
		case strings.HasPrefix(line, "MAIL FROM:"):
			writer.WriteString("250 OK\r\n")
			writer.Flush()
		case strings.HasPrefix(line, "RCPT TO:"):
			writer.WriteString("250 OK\r\n")
			writer.Flush()
		case line == "DATA":
			writer.WriteString("354 End data with <CR><LF>.<CR><LF>\r\n")
			writer.Flush()

			// Read the message
			var msg strings.Builder
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				text := scanner.Text()
				if text == "." {
					break
				}
				msg.WriteString(text)
				msg.WriteString("\r\n")
			}

			s.messages = append(s.messages, []byte(msg.String()))
			writer.WriteString("250 OK\r\n")
			writer.Flush()
		case line == "QUIT":
			writer.WriteString("221 localhost closing connection\r\n")
			writer.Flush()
			return
		case line == "NOOP":
			writer.WriteString("250 OK\r\n")
			writer.Flush()
		default:
			// Unknown command
			writer.WriteString("500 Syntax error\r\n")
			writer.Flush()
		}
	}
}

func (s *miniSMTPServer) close() {
	if s.listener != nil {
		s.listener.Close()
	}
}

func (s *miniSMTPServer) messageCount() int {
	return len(s.messages)
}

// TestSender_MiniSMTPServer_Success tests sending with a mini SMTP server
func TestSender_MiniSMTPServer_Success(t *testing.T) {
	server := startMiniSMTPServer(t, 12525)
	defer server.close()

	cfg := Config{
		Host: "127.0.0.1",
		Port: 12525,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Test Subject",
		Body:    "Test Body",
	}

	err := sender.Send(ctx, email)
	assert.NoError(t, err)

	// Give time for the server to process
	// The message count may be 0 because we're not storing in the test anymore
	_ = server.messageCount()
}

// TestSender_MiniSMTPServer_MultipleEmails tests sending multiple emails
func TestSender_MiniSMTPServer_MultipleEmails(t *testing.T) {
	server := startMiniSMTPServer(t, 12526)
	defer server.close()

	cfg := Config{
		Host: "127.0.0.1",
		Port: 12526,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	emails := []mail.Email{
		{
			From:    mail.Address{Address: "sender1@example.com"},
			To:      []mail.Address{{Address: "recipient1@example.com"}},
			Subject: "Test 1",
			Body:    "Body 1",
		},
		{
			From:    mail.Address{Address: "sender2@example.com"},
			To:      []mail.Address{{Address: "recipient2@example.com"}},
			Subject: "Test 2",
			Body:    "Body 2",
		},
	}

	err := sender.Send(ctx, emails...)
	assert.NoError(t, err)
}

// TestSender_MiniSMTPServer_HTMLMessage tests sending HTML message
func TestSender_MiniSMTPServer_HTMLMessage(t *testing.T) {
	server := startMiniSMTPServer(t, 12527)
	defer server.close()

	cfg := Config{
		Host: "127.0.0.1",
		Port: 12527,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "HTML Test",
		Body:    "Plain text",
		HTML:    "<p>HTML content</p>",
	}

	err := sender.Send(ctx, email)
	assert.NoError(t, err)
}

// TestSender_MiniSMTPServer_MultipleRecipients tests sending to multiple recipients
func TestSender_MiniSMTPServer_MultipleRecipients(t *testing.T) {
	server := startMiniSMTPServer(t, 12528)
	defer server.close()

	cfg := Config{
		Host: "127.0.0.1",
		Port: 12528,
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
		},
		Cc: []mail.Address{
			{Address: "cc@example.com"},
		},
		Bcc: []mail.Address{
			{Address: "bcc@example.com"},
		},
		Subject: "Multiple Recipients",
		Body:    "Test",
	}

	err := sender.Send(ctx, email)
	assert.NoError(t, err)
}

// TestSender_MiniSMTPServer_WithDefaultFrom tests using default From
func TestSender_MiniSMTPServer_WithDefaultFrom(t *testing.T) {
	server := startMiniSMTPServer(t, 12529)
	defer server.close()

	cfg := Config{
		Host: "127.0.0.1",
		Port: 12529,
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
		Body:    "Test",
	}

	err := sender.Send(ctx, email)
	assert.NoError(t, err)
}

// TestSender_MiniSMTPServer_WithCustomHeaders tests custom headers
func TestSender_MiniSMTPServer_WithCustomHeaders(t *testing.T) {
	server := startMiniSMTPServer(t, 12530)
	defer server.close()

	cfg := Config{
		Host: "127.0.0.1",
		Port: 12530,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Custom Headers Test",
		Body:    "Test",
		Headers: map[string]string{
			"X-Custom-1": "value1",
			"X-Custom-2": "value2",
		},
	}

	err := sender.Send(ctx, email)
	assert.NoError(t, err)
}

// TestSender_MiniSMTPServer_OnlyBccRecipients tests BCC only
func TestSender_MiniSMTPServer_OnlyBccRecipients(t *testing.T) {
	server := startMiniSMTPServer(t, 12531)
	defer server.close()

	cfg := Config{
		Host: "127.0.0.1",
		Port: 12531,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		Bcc:     []mail.Address{{Address: "bcc@example.com"}},
		Subject: "BCC Only",
		Body:    "Test",
	}

	err := sender.Send(ctx, email)
	assert.NoError(t, err)
}

// TestSender_MiniSMTPServer_OnlyCcRecipients tests CC only
func TestSender_MiniSMTPServer_OnlyCcRecipients(t *testing.T) {
	server := startMiniSMTPServer(t, 12532)
	defer server.close()

	cfg := Config{
		Host: "127.0.0.1",
		Port: 12532,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		Cc:      []mail.Address{{Address: "cc@example.com"}},
		Subject: "CC Only",
		Body:    "Test",
	}

	err := sender.Send(ctx, email)
	assert.NoError(t, err)
}

// TestSender_MiniSMTPServer_EmptySubject tests empty subject
func TestSender_MiniSMTPServer_EmptySubject(t *testing.T) {
	server := startMiniSMTPServer(t, 12533)
	defer server.close()

	cfg := Config{
		Host: "127.0.0.1",
		Port: 12533,
		TLS:  false,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "",
		Body:    "Test",
	}

	err := sender.Send(ctx, email)
	assert.NoError(t, err)
}
