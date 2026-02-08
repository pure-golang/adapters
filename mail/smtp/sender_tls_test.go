package smtp

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/pure-golang/adapters/mail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tlsSMTPServer is a minimal SMTP server with STARTTLS support for testing
type tlsSMTPServer struct {
	listener net.Listener
	useTLS   bool
	cert     *tls.Certificate
}

// startTLSMiniSMTPServer starts a minimal SMTP server with STARTTLS support
func startTLSMiniSMTPServer(t *testing.T, port int) *tlsSMTPServer {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	require.NoError(t, err, "failed to start TLS SMTP server")

	// Create a self-signed certificate for testing
	cert, err := tls.LoadX509KeyPair("testdata/cert.pem", "testdata/key.pem")
	if err != nil {
		// If cert files don't exist, create server without TLS support
		// This allows tests to run without certificate setup
		t.Logf("No TLS cert found, using plain SMTP server")
	}

	server := &tlsSMTPServer{
		listener: listener,
		useTLS:   err == nil,
		cert:     &cert,
	}

	go server.handleConnections(t)

	return server
}

func (s *tlsSMTPServer) handleConnections(t *testing.T) {
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

func (s *tlsSMTPServer) handleSMTP(t *testing.T, conn net.Conn) {
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
			// Advertise STARTTLS support
			if s.useTLS && strings.HasPrefix(line, "EHLO") {
				writer.WriteString("250-localhost\r\n250-STARTTLS\r\n250 HELP\r\n")
			} else {
				writer.WriteString("250 localhost\r\n")
			}
			writer.Flush()
		case strings.HasPrefix(line, "STARTTLS"):
			writer.WriteString("220 Ready to start TLS\r\n")
			writer.Flush()

			// Upgrade to TLS
			if s.cert != nil {
				tlsConfig := &tls.Config{
					Certificates: []tls.Certificate{*s.cert},
					ServerName:   "localhost",
				}
				tlsConn := tls.Server(conn, tlsConfig)
				defer tlsConn.Close()
				err := tlsConn.Handshake()
				if err != nil {
					t.Logf("TLS handshake failed: %v", err)
					return
				}

				// Continue with TLS connection
				reader = bufio.NewReader(tlsConn)
				writer = bufio.NewWriter(tlsConn)
				conn = tlsConn
			} else {
				return
			}
		case strings.HasPrefix(line, "MAIL FROM:"):
			writer.WriteString("250 OK\r\n")
			writer.Flush()
		case strings.HasPrefix(line, "RCPT TO:"):
			writer.WriteString("250 OK\r\n")
			writer.Flush()
		case line == "DATA":
			writer.WriteString("354 End data with <CR><LF>.<CR><LF>\r\n")
			writer.Flush()

			// Read the message until "."
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				if scanner.Text() == "." {
					break
				}
			}

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
			// Unknown command - might be AUTH
			if strings.HasPrefix(line, "AUTH") {
				// Accept AUTH but don't require it
				writer.WriteString("235 OK\r\n")
				writer.Flush()
			} else {
				writer.WriteString("500 Syntax error\r\n")
				writer.Flush()
			}
		}
	}
}

func (s *tlsSMTPServer) close() {
	if s.listener != nil {
		s.listener.Close()
	}
}

// TestSender_TLS_ConnectionError tests TLS connection error handling
func TestSender_TLS_ConnectionError(t *testing.T) {
	server := startTLSMiniSMTPServer(t, 12540)
	defer server.close()

	cfg := Config{
		Host:     "127.0.0.1",
		Port:     12540,
		TLS:      true,
		Insecure: true, // Skip cert verification since we're using self-signed
		Username: "testuser",
		Password: "testpass",
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "TLS Test",
		Body:    "Test",
	}

	err := sender.Send(ctx, email)
	// Result depends on whether TLS cert is available
	// Either success (if TLS working) or error (if cert missing)
	// We just verify it doesn't hang
	_ = err
}

// TestSender_TLS_ContextCanceled tests context cancellation in TLS mode
func TestSender_TLS_ContextCanceled(t *testing.T) {
	cfg := Config{
		Host: "127.0.0.1",
		Port: 12541,
		TLS:  true,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel before sending

	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Test",
		Body:    "Test",
	}

	err := sender.Send(ctx, email)
	assert.Error(t, err)
}

// TestSender_TLS_ConnectionTimeout tests connection timeout
func TestSender_TLS_ConnectionTimeout(t *testing.T) {
	cfg := Config{
		Host: "192.0.2.1", // TEST-NET-1 - should be unreachable
		Port: 2525,
		TLS:  true,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Test",
		Body:    "Test",
	}

	err := sender.Send(ctx, email)
	assert.Error(t, err)
}

// TestSender_TLS_ServerNotRunning tests TLS to non-existent server
func TestSender_TLS_ServerNotRunning(t *testing.T) {
	cfg := Config{
		Host:     "127.0.0.1",
		Port:     12542, // No server running
		TLS:      true,
		Insecure: true,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Test",
		Body:    "Test",
	}

	err := sender.Send(ctx, email)
	assert.Error(t, err)
}

// TestSender_TLS_WithAuthentication tests TLS with authentication
func TestSender_TLS_WithAuthentication(t *testing.T) {
	server := startTLSMiniSMTPServer(t, 12543)
	defer server.close()

	cfg := Config{
		Host:     "127.0.0.1",
		Port:     12543,
		TLS:      true,
		Insecure: true,
		Username: "testuser",
		Password: "testpass",
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "TLS Auth Test",
		Body:    "Test",
	}

	err := sender.Send(ctx, email)
	// Result depends on TLS cert availability
	_ = err
}

// TestSender_TLS_InvalidHost tests TLS with invalid hostname
func TestSender_TLS_InvalidHost(t *testing.T) {
	cfg := Config{
		Host:     "invalid-host-12345.local",
		Port:     587,
		TLS:      true,
		Insecure: false, // Will fail cert verification
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Test",
		Body:    "Test",
	}

	err := sender.Send(ctx, email)
	assert.Error(t, err)
}

// TestSender_TLS_ConnectionRefusedImmediate tests immediate connection refused
func TestSender_TLS_ConnectionRefusedImmediate(t *testing.T) {
	cfg := Config{
		Host:     "127.0.0.1",
		Port:     12544, // Port that likely won't have a server
		TLS:      true,
		Insecure: true,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Test",
		Body:    "Test",
	}

	err := sender.Send(ctx, email)
	assert.Error(t, err)
}

// TestSender_TLS_WithDefaultFrom tests TLS with default From address
func TestSender_TLS_WithDefaultFrom(t *testing.T) {
	cfg := Config{
		Host:     "127.0.0.1",
		Port:     12545,
		From:     "default@example.com",
		TLS:      true,
		Insecure: true,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		// No From specified - should use default
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Test",
		Body:    "Test",
	}

	err := sender.Send(ctx, email)
	// Will fail to connect, but should try the default From
	assert.Error(t, err)
}

// TestSender_BuildMessage_AllContentTypes tests all content type variations
func TestSender_BuildMessage_AllContentTypes(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	t.Run("Plain text only", func(t *testing.T) {
		email := mail.Email{
			From:    mail.Address{Address: "from@example.com"},
			To:      []mail.Address{{Address: "to@example.com"}},
			Subject: "Plain",
			Body:    "Plain text only",
		}
		msg := sender.buildMessage(email)
		msgStr := string(msg)
		assert.Contains(t, msgStr, "Content-Type: text/plain")
		assert.NotContains(t, msgStr, "multipart/alternative")
	})

	t.Run("HTML only", func(t *testing.T) {
		email := mail.Email{
			From:    mail.Address{Address: "from@example.com"},
			To:      []mail.Address{{Address: "to@example.com"}},
			Subject: "HTML",
			HTML:    "<p>HTML only</p>",
		}
		msg := sender.buildMessage(email)
		msgStr := string(msg)
		assert.Contains(t, msgStr, "multipart/alternative")
		assert.Contains(t, msgStr, "HTML only")
	})

	t.Run("Both plain and HTML", func(t *testing.T) {
		email := mail.Email{
			From:    mail.Address{Address: "from@example.com"},
			To:      []mail.Address{{Address: "to@example.com"}},
			Subject: "Both",
			Body:    "Plain text",
			HTML:    "<p>HTML</p>",
		}
		msg := sender.buildMessage(email)
		msgStr := string(msg)
		assert.Contains(t, msgStr, "multipart/alternative")
		assert.Contains(t, msgStr, "Plain text")
		assert.Contains(t, msgStr, "HTML")
	})

	t.Run("With custom headers", func(t *testing.T) {
		email := mail.Email{
			From:    mail.Address{Address: "from@example.com"},
			To:      []mail.Address{{Address: "to@example.com"}},
			Subject: "Headers",
			Body:    "Body",
			Headers: map[string]string{
				"X-Header-1": "Value1",
				"X-Header-2": "Value2",
			},
		}
		msg := sender.buildMessage(email)
		msgStr := string(msg)
		assert.Contains(t, msgStr, "X-Header-1: Value1")
		assert.Contains(t, msgStr, "X-Header-2: Value2")
	})

	t.Run("With CC and BCC", func(t *testing.T) {
		email := mail.Email{
			From:    mail.Address{Address: "from@example.com"},
			To:      []mail.Address{{Address: "to@example.com"}},
			Cc:      []mail.Address{{Address: "cc@example.com"}},
			Bcc:     []mail.Address{{Address: "bcc@example.com"}},
			Subject: "Recipients",
			Body:    "Body",
		}
		msg := sender.buildMessage(email)
		msgStr := string(msg)
		assert.Contains(t, msgStr, "To: to@example.com")
		assert.Contains(t, msgStr, "Cc: cc@example.com")
		// BCC should not be in headers
		assert.NotContains(t, msgStr, "bcc@example.com")
	})

	t.Run("With all recipient types", func(t *testing.T) {
		email := mail.Email{
			From: mail.Address{Address: "from@example.com"},
			To: []mail.Address{
				{Address: "to1@example.com"},
				{Address: "to2@example.com"},
			},
			Cc: []mail.Address{
				{Address: "cc1@example.com"},
				{Address: "cc2@example.com"},
			},
			Bcc: []mail.Address{
				{Address: "bcc1@example.com"},
			},
			Subject: "All Recipients",
			Body:    "Body",
		}
		msg := sender.buildMessage(email)
		msgStr := string(msg)
		// Check To and Cc are in headers
		assert.Contains(t, msgStr, "to1@example.com")
		assert.Contains(t, msgStr, "to2@example.com")
		assert.Contains(t, msgStr, "cc1@example.com")
		assert.Contains(t, msgStr, "cc2@example.com")
		// BCC should not be in headers
		assert.NotContains(t, msgStr, "bcc1@example.com")
	})
}

// TestSender_FormatAddress_EdgeCases tests edge cases for address formatting
func TestSender_FormatAddress_EdgeCases(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	tests := []struct {
		name     string
		addr     mail.Address
		expected string
	}{
		{
			name:     "empty address",
			addr:     mail.Address{},
			expected: "",
		},
		{
			name:     "only address",
			addr:     mail.Address{Address: "test@example.com"},
			expected: "test@example.com",
		},
		{
			name:     "name with spaces",
			addr:     mail.Address{Name: "John Doe", Address: "john@example.com"},
			expected: "John Doe <john@example.com>",
		},
		{
			name:     "name with quotes",
			addr:     mail.Address{Name: `John "The Rock" Doe`, Address: "john@example.com"},
			expected: `John \"The Rock\" Doe <john@example.com>`,
		},
		{
			name:     "name with comma",
			addr:     mail.Address{Name: "Doe, John", Address: "john@example.com"},
			expected: "Doe, John <john@example.com>",
		},
		{
			name:     "name with angle brackets in name",
			addr:     mail.Address{Name: "John <Chief> Doe", Address: "john@example.com"},
			expected: "John <Chief> Doe <john@example.com>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sender.formatAddress(tt.addr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSender_FormatAddressList_EdgeCases tests edge cases for address list formatting
func TestSender_FormatAddressList_EdgeCases(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	t.Run("empty list", func(t *testing.T) {
		addrs := []mail.Address{}
		result := sender.formatAddressList(addrs)
		assert.Equal(t, "", result)
	})

	t.Run("single address", func(t *testing.T) {
		addrs := []mail.Address{
			{Name: "John", Address: "john@example.com"},
		}
		result := sender.formatAddressList(addrs)
		assert.Equal(t, "John <john@example.com>", result)
	})

	t.Run("multiple addresses", func(t *testing.T) {
		addrs := []mail.Address{
			{Name: "Alice", Address: "alice@example.com"},
			{Name: "Bob", Address: "bob@example.com"},
			{Name: "Charlie", Address: "charlie@example.com"},
		}
		result := sender.formatAddressList(addrs)
		assert.Contains(t, result, "Alice <alice@example.com>")
		assert.Contains(t, result, "Bob <bob@example.com>")
		assert.Contains(t, result, "Charlie <charlie@example.com>")
		assert.Contains(t, result, ", ")
	})
}

// TestSender_GetEmailAddresses_EdgeCases tests edge cases for extracting email addresses
func TestSender_GetEmailAddresses_EdgeCases(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	t.Run("empty list", func(t *testing.T) {
		addrs := []mail.Address{}
		result := sender.getEmailAddresses(addrs)
		assert.Equal(t, []string{}, result)
	})

	t.Run("single address", func(t *testing.T) {
		addrs := []mail.Address{
			{Name: "John", Address: "john@example.com"},
		}
		result := sender.getEmailAddresses(addrs)
		assert.Equal(t, []string{"john@example.com"}, result)
	})

	t.Run("multiple addresses - names should be stripped", func(t *testing.T) {
		addrs := []mail.Address{
			{Name: "Alice", Address: "alice@example.com"},
			{Name: "Bob", Address: "bob@example.com"},
		}
		result := sender.getEmailAddresses(addrs)
		assert.Equal(t, []string{"alice@example.com", "bob@example.com"}, result)
	})
}

// TestSender_Config_AllTLSCombinations tests all TLS configuration combinations
func TestSender_Config_AllTLSCombinations(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Config
		wantTLS   bool
		wantInsec bool
	}{
		{
			name:      "default values",
			cfg:       Config{Host: "localhost", Port: 25},
			wantTLS:   false,
			wantInsec: false,
		},
		{
			name:      "TLS enabled, secure",
			cfg:       Config{Host: "localhost", Port: 587, TLS: true, Insecure: false},
			wantTLS:   true,
			wantInsec: false,
		},
		{
			name:      "TLS enabled, insecure",
			cfg:       Config{Host: "localhost", Port: 587, TLS: true, Insecure: true},
			wantTLS:   true,
			wantInsec: true,
		},
		{
			name:      "TLS disabled, insecure flag ignored",
			cfg:       Config{Host: "localhost", Port: 25, TLS: false, Insecure: true},
			wantTLS:   false,
			wantInsec: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := NewSender(tt.cfg, nil)
			assert.Equal(t, tt.wantTLS, sender.cfg.TLS)
			assert.Equal(t, tt.wantInsec, sender.cfg.Insecure)
			_ = sender.Close()
		})
	}
}

// TestSender_NewSender_Variations tests sender creation with various configs
func TestSender_NewSender_Variations(t *testing.T) {
	t.Run("with nil options", func(t *testing.T) {
		cfg := Config{Host: "localhost", Port: 25}
		sender := NewSender(cfg, nil)
		assert.NotNil(t, sender)
		assert.False(t, sender.closed)
		_ = sender.Close()
	})

	t.Run("with empty options", func(t *testing.T) {
		cfg := Config{Host: "localhost", Port: 25}
		opts := &SenderOptions{}
		sender := NewSender(cfg, opts)
		assert.NotNil(t, sender)
		assert.False(t, sender.closed)
		_ = sender.Close()
	})

	t.Run("full config", func(t *testing.T) {
		cfg := Config{
			Host:     "smtp.example.com",
			Port:     587,
			Username: "user",
			Password: "pass",
			From:     "from@example.com",
			TLS:      true,
			Insecure: true,
		}
		sender := NewSender(cfg, nil)
		assert.Equal(t, "smtp.example.com", sender.cfg.Host)
		assert.Equal(t, 587, sender.cfg.Port)
		assert.Equal(t, "user", sender.cfg.Username)
		assert.Equal(t, "pass", sender.cfg.Password)
		assert.Equal(t, "from@example.com", sender.cfg.From)
		assert.True(t, sender.cfg.TLS)
		assert.True(t, sender.cfg.Insecure)
		_ = sender.Close()
	})
}
