package mail

import (
	"context"
	"io"
)

// Sender sends emails via SMTP.
type Sender interface {
	Send(ctx context.Context, emails ...Email) error
	io.Closer
}

// Email represents an email message.
type Email struct {
	// Envelope
	From    Address
	To      []Address
	Cc      []Address
	Bcc     []Address
	Subject string

	// Headers
	Headers map[string]string

	// Body
	Body string // Plain text body
	HTML string // HTML body (optional)
}

// Address represents an email address.
type Address struct {
	Name    string // "John Doe"
	Address string // "john@example.com"
}
