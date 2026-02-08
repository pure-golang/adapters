package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"github.com/pure-golang/adapters/mail"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var _ mail.Sender = (*Sender)(nil)

// Sender implements mail.Sender using net/smtp.
type Sender struct {
	mx     sync.Mutex
	cfg    Config
	closed bool
}

// SenderOptions contains options for creating a Sender.
type SenderOptions struct {
	// Logger can be added later if needed
}

// NewSender creates a new SMTP Sender.
func NewSender(cfg Config, options *SenderOptions) *Sender {
	return &Sender{
		cfg:    cfg,
		closed: false,
	}
}

// Send sends one or more emails.
func (s *Sender) Send(ctx context.Context, emails ...mail.Email) error {
	for _, email := range emails {
		if err := s.send(ctx, email); err != nil {
			return err
		}
	}
	return nil
}

// send sends a single email.
func (s *Sender) send(ctx context.Context, email mail.Email) error {
	ctx, span := tracer.Start(ctx, "SMTP.Send", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	// Set span attributes
	span.SetAttributes(
		attribute.String("smtp.from", email.From.Address),
		attribute.String("smtp.subject", email.Subject),
		attribute.Int("smtp.to_count", len(email.To)),
		attribute.Int("smtp.cc_count", len(email.Cc)),
		attribute.Int("smtp.bcc_count", len(email.Bcc)),
		attribute.String("smtp.host", s.cfg.Host),
		attribute.Int("smtp.port", s.cfg.Port),
		attribute.Bool("smtp.tls", s.cfg.TLS),
	)

	s.mx.Lock()
	defer s.mx.Unlock()

	if s.closed {
		span.SetStatus(codes.Error, "sender is closed")
		return errors.New("sender is closed")
	}

	// Build email content
	from := email.From.Address
	if from == "" {
		from = s.cfg.From
	}
	if from == "" {
		return errors.New("no from address specified")
	}

	toAddresses := s.getEmailAddresses(email.To)
	ccAddresses := s.getEmailAddresses(email.Cc)
	bccAddresses := s.getEmailAddresses(email.Bcc)

	if len(toAddresses) == 0 && len(ccAddresses) == 0 && len(bccAddresses) == 0 {
		return errors.New("no recipients specified")
	}

	// Build message
	msg := s.buildMessage(email)

	// SMTP server address
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	var err error
	if s.cfg.TLS {
		err = s.sendWithTLS(ctx, addr, auth, from, append(toAddresses, ccAddresses...), bccAddresses, msg)
	} else {
		err = smtp.SendMail(addr, auth, from, append(toAddresses, ccAddresses...), msg)
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return errors.Wrap(err, "failed to send email")
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// sendWithTLS sends email using STARTTLS.
func (s *Sender) sendWithTLS(ctx context.Context, addr string, auth smtp.Auth, from string, to, bcc []string, msg []byte) error {
	ctx, span := tracer.Start(ctx, "SMTP.SendWithTLS")
	defer span.End()

	span.SetAttributes(
		attribute.String("smtp.address", addr),
		attribute.String("smtp.from", from),
		attribute.Int("smtp.recipients_count", len(to)+len(bcc)),
		attribute.Bool("smtp.auth", auth != nil),
	)

	// Check for context cancellation
	select {
	case <-ctx.Done():
		span.SetStatus(codes.Error, "context canceled")
		return ctx.Err()
	default:
	}

	// Connect to server
	client, err := smtp.Dial(addr)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to connect")
		return errors.Wrap(err, "failed to connect to SMTP server")
	}
	defer func() {
		if err := client.Close(); err != nil {
			// Error closing SMTP connection is not critical here as the message has already been sent.
			// The connection will be cleaned up by the server.
			_ = err // Explicitly ignore to satisfy linters
		}
	}()

	// Start TLS if available
	if ok, _ := client.Extension("STARTTLS"); ok {
		span.SetAttributes(attribute.Bool("smtp.starttls", true))

		tlsConfig := &tls.Config{
			ServerName:         s.cfg.Host,
			InsecureSkipVerify: s.cfg.Insecure, // #nosec G402 -- controlled by config, user's responsibility
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to start TLS")
			return errors.Wrap(err, "failed to start TLS")
		}
	} else {
		span.SetAttributes(attribute.Bool("smtp.starttls", false))
	}

	// Authenticate if credentials provided
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to authenticate")
			return errors.Wrap(err, "failed to authenticate")
		}
	}

	// Set sender
	if err := client.Mail(from); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to set sender")
		return errors.Wrap(err, "failed to set sender")
	}

	// Set recipients
	allRecipients := append(to, bcc...)
	for _, addr := range allRecipients {
		if err := client.Rcpt(addr); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "failed to set recipient")
			return errors.Wrapf(err, "failed to set recipient: %s", addr)
		}
	}

	// Send data
	writer, err := client.Data()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get data writer")
		return errors.Wrap(err, "failed to get data writer")
	}
	defer func() {
		if err := writer.Close(); err != nil {
			// Error closing data writer is not critical as the message has already been sent.
			_ = err // Explicitly ignore to satisfy linters
		}
	}()

	_, err = writer.Write(msg)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to write message")
		return errors.Wrap(err, "failed to write message")
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// buildMessage builds the raw email message.
func (s *Sender) buildMessage(email mail.Email) []byte {
	var msg strings.Builder

	// Headers
	msg.WriteString(fmt.Sprintf("From: %s\r\n", s.formatAddress(email.From)))

	if len(email.To) > 0 {
		msg.WriteString(fmt.Sprintf("To: %s\r\n", s.formatAddressList(email.To)))
	}

	if len(email.Cc) > 0 {
		msg.WriteString(fmt.Sprintf("Cc: %s\r\n", s.formatAddressList(email.Cc)))
	}

	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))

	// Add custom headers
	for k, v := range email.Headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}

	// Build body
	if email.HTML != "" {
		boundary := fmt.Sprintf("boundary_%d", time.Now().UnixNano())
		msg.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s\r\n", boundary))
		msg.WriteString("\r\n")

		// Plain text part
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		msg.WriteString(email.Body)
		msg.WriteString("\r\n")

		// HTML part
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		msg.WriteString(email.HTML)
		msg.WriteString("\r\n")

		msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else {
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		msg.WriteString(email.Body)
		msg.WriteString("\r\n")
	}

	return []byte(msg.String())
}

// formatAddress formats a single address.
func (s *Sender) formatAddress(addr mail.Address) string {
	if addr.Name != "" {
		// Escape quotes in name
		escapedName := strings.ReplaceAll(addr.Name, "\"", "\\\"")
		return fmt.Sprintf("%s <%s>", escapedName, addr.Address)
	}
	return addr.Address
}

// formatAddressList formats a list of addresses.
func (s *Sender) formatAddressList(addrs []mail.Address) string {
	formatted := make([]string, len(addrs))
	for i, addr := range addrs {
		formatted[i] = s.formatAddress(addr)
	}
	return strings.Join(formatted, ", ")
}

// getEmailAddresses extracts email addresses from mail.Address slice.
func (s *Sender) getEmailAddresses(addrs []mail.Address) []string {
	result := make([]string, len(addrs))
	for i, addr := range addrs {
		result[i] = addr.Address
	}
	return result
}

// Close closes the sender.
func (s *Sender) Close() error {
	s.mx.Lock()
	defer s.mx.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true
	return nil
}
