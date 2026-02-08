package smtp

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/pure-golang/adapters/mail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestCert generates a self-signed certificate for testing
func generateTestCert() (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test SMTP"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return tls.Certificate{}, err
	}

	return cert, nil
}

// starttlssmtpServer is a minimal SMTP server that supports STARTTLS
type starttlssmtpServer struct {
	listener net.Listener
	cert     tls.Certificate
}

// starttlssmtpServer creates and starts a STARTTLS-enabled SMTP server
func startSTARTTLSServer(t *testing.T, port int) *starttlssmtpServer {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	require.NoError(t, err, "failed to listen")

	cert, err := generateTestCert()
	require.NoError(t, err, "failed to generate cert")

	server := &starttlssmtpServer{
		listener: listener,
		cert:     cert,
	}

	go server.run(t)

	return server
}

func (s *starttlssmtpServer) run(t *testing.T) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(t, conn)
	}
}

func (s *starttlssmtpServer) handleConn(t *testing.T, conn net.Conn) {
	defer conn.Close()

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
		case strings.ToUpper(line) == "EHLO localhost" || strings.HasPrefix(strings.ToUpper(line), "EHLO"):
			// Respond with EHLO and advertise STARTTLS
			writer.WriteString("250-localhost\r\n")
			writer.WriteString("250-STARTTLS\r\n")
			writer.WriteString("250 HELP\r\n")
			writer.Flush()
		case strings.ToUpper(line) == "STARTTLS":
			// Respond to STARTTLS
			writer.WriteString("220 Ready to start TLS\r\n")
			writer.Flush()

			// Upgrade to TLS
			tlsConfig := &tls.Config{
				Certificates: []tls.Certificate{s.cert},
				ServerName:   "localhost",
				MinVersion:   tls.VersionTLS12,
			}
			tlsConn := tls.Server(conn, tlsConfig)

			if err := tlsConn.Handshake(); err != nil {
				t.Logf("TLS handshake failed: %v", err)
				return
			}

			// Continue with TLS connection
			reader = bufio.NewReader(tlsConn)
			writer = bufio.NewWriter(tlsConn)
			conn = tlsConn

		case strings.HasPrefix(strings.ToUpper(line), "MAIL FROM:"):
			writer.WriteString("250 OK\r\n")
			writer.Flush()
		case strings.HasPrefix(strings.ToUpper(line), "RCPT TO:"):
			writer.WriteString("250 OK\r\n")
			writer.Flush()
		case line == "DATA":
			writer.WriteString("354 End data with <CR><LF>.<CR><LF>\r\n")
			writer.Flush()

			// Read message
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				if scanner.Text() == "." {
					break
				}
			}

			writer.WriteString("250 OK\r\n")
			writer.Flush()
		case strings.HasPrefix(strings.ToUpper(line), "AUTH PLAIN"):
			// Accept AUTH
			writer.WriteString("235 OK\r\n")
			writer.Flush()
		case strings.ToUpper(line) == "QUIT":
			writer.WriteString("221 Bye\r\n")
			writer.Flush()
			return
		default:
			// Unknown command - try to handle gracefully
			writer.WriteString("500 Syntax error\r\n")
			writer.Flush()
		}
	}
}

func (s *starttlssmtpServer) close() {
	if s.listener != nil {
		s.listener.Close()
	}
}

// TestSender_STARTTLS_Success tests successful STARTTLS connection
func TestSender_STARTTLS_Success(t *testing.T) {
	server := startSTARTTLSServer(t, 12550)
	defer server.close()

	cfg := Config{
		Host:     "127.0.0.1",
		Port:     12550,
		TLS:      true,
		Insecure: true, // Skip cert verification
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "STARTTLS Test",
		Body:    "Test email with STARTTLS",
	}

	err := sender.Send(ctx, email)
	assert.NoError(t, err)
}

// TestSender_STARTTLS_WithAuth tests STARTTLS with authentication
func TestSender_STARTTLS_WithAuth(t *testing.T) {
	server := startSTARTTLSServer(t, 12551)
	defer server.close()

	cfg := Config{
		Host:     "127.0.0.1",
		Port:     12551,
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
		Subject: "STARTTLS Auth Test",
		Body:    "Test email with STARTTLS and AUTH",
	}

	err := sender.Send(ctx, email)
	assert.NoError(t, err)
}

// TestSender_STARTTLS_MultipleRecipients tests STARTTLS with multiple recipients
func TestSender_STARTTLS_MultipleRecipients(t *testing.T) {
	server := startSTARTTLSServer(t, 12552)
	defer server.close()

	cfg := Config{
		Host:     "127.0.0.1",
		Port:     12552,
		TLS:      true,
		Insecure: true,
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
		Subject: "Multiple Recipients STARTTLS",
		Body:    "Test",
	}

	err := sender.Send(ctx, email)
	assert.NoError(t, err)
}

// TestSender_STARTTLS_WithHTML tests STARTTLS with HTML email
func TestSender_STARTTLS_WithHTML(t *testing.T) {
	server := startSTARTTLSServer(t, 12553)
	defer server.close()

	cfg := Config{
		Host:     "127.0.0.1",
		Port:     12553,
		TLS:      true,
		Insecure: true,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "HTML STARTTLS Test",
		Body:    "Plain text",
		HTML:    "<html><body>HTML content</body></html>",
	}

	err := sender.Send(ctx, email)
	assert.NoError(t, err)
}

// TestSender_STARTTLS_ContextCancellation tests context cancellation during STARTTLS
func TestSender_STARTTLS_ContextCancellation(t *testing.T) {
	server := startSTARTTLSServer(t, 12554)
	defer server.close()

	cfg := Config{
		Host:     "127.0.0.1",
		Port:     12554,
		TLS:      true,
		Insecure: true,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Test",
		Body:    "Test",
	}

	err := sender.Send(ctx, email)
	assert.Error(t, err)
}

// TestSender_STARTTLS_WithDefaultFrom tests STARTTLS with default From
func TestSender_STARTTLS_WithDefaultFrom(t *testing.T) {
	server := startSTARTTLSServer(t, 12555)
	defer server.close()

	cfg := Config{
		Host:     "127.0.0.1",
		Port:     12555,
		From:     "default@example.com",
		TLS:      true,
		Insecure: true,
	}

	sender := NewSender(cfg, nil)
	defer sender.Close()

	ctx := context.Background()
	email := mail.Email{
		// No From - should use default
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Default From STARTTLS",
		Body:    "Test",
	}

	err := sender.Send(ctx, email)
	assert.NoError(t, err)
}
