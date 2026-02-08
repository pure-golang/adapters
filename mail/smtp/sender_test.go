package smtp

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pure-golang/adapters/mail"
)

func TestNewSender_WithDefaultConfig(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     2525,
		Username: "test",
		Password: "test",
		TLS:      true, // explicit for test
	}

	sender := NewSender(cfg, nil)

	assert.NotNil(t, sender)
	assert.Equal(t, "localhost", sender.cfg.Host)
	assert.Equal(t, 2525, sender.cfg.Port)
	assert.True(t, sender.cfg.TLS)

	err := sender.Close()
	assert.NoError(t, err)
}

func TestSender_CloseTwice(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     2525,
		Username: "test",
		Password: "test",
	}

	sender := NewSender(cfg, nil)

	err := sender.Close()
	assert.NoError(t, err)

	err = sender.Close()
	assert.NoError(t, err)
}

func TestSender_Send_WhenClosed(t *testing.T) {
	cfg := Config{
		Host: "localhost",
		Port: 2525,
	}
	sender := NewSender(cfg, nil)

	require.NoError(t, sender.Close())

	ctx := context.Background()
	email := mail.Email{
		From: mail.Address{Address: "sender@example.com"},
		To:   []mail.Address{{Address: "recipient@example.com"}},
	}

	err := sender.Send(ctx, email)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestSender_Send_NoFromAddress(t *testing.T) {
	cfg := Config{
		Host: "localhost",
		Port: 2525,
	}
	sender := NewSender(cfg, nil)

	ctx := context.Background()
	email := mail.Email{
		To: []mail.Address{{Address: "recipient@example.com"}},
	}

	err := sender.Send(ctx, email)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no from address")
}

func TestSender_Send_NoRecipients(t *testing.T) {
	cfg := Config{
		Host: "localhost",
		Port: 2525,
	}
	sender := NewSender(cfg, nil)

	ctx := context.Background()
	email := mail.Email{
		From: mail.Address{Address: "sender@example.com"},
	}

	err := sender.Send(ctx, email)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no recipients")
}

func TestSender_BuildMessage(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	email := mail.Email{
		From:    mail.Address{Name: "Sender", Address: "sender@example.com"},
		To:      []mail.Address{{Name: "Recipient", Address: "recipient@example.com"}},
		Subject: "Test Subject",
		Body:    "Test Body",
		Headers: map[string]string{"X-Custom": "value"},
	}

	msg := sender.buildMessage(email)

	msgStr := string(msg)
	assert.Contains(t, msgStr, "From: Sender <sender@example.com>")
	assert.Contains(t, msgStr, "To: Recipient <recipient@example.com>")
	assert.Contains(t, msgStr, "Subject: Test Subject")
	assert.Contains(t, msgStr, "X-Custom: value")
	assert.Contains(t, msgStr, "Test Body")
	assert.Contains(t, msgStr, "MIME-Version: 1.0")
}

func TestSender_BuildMessageWithHTML(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "HTML Test",
		Body:    "Plain text",
		HTML:    "<p>HTML content</p>",
	}

	msg := sender.buildMessage(email)

	msgStr := string(msg)
	assert.Contains(t, msgStr, "multipart/alternative")
	assert.Contains(t, msgStr, "Plain text")
	assert.Contains(t, msgStr, "<p>HTML content</p>")
	assert.Contains(t, msgStr, "boundary_")
}

func TestSender_BuildMessageWithCcAndBcc(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	email := mail.Email{
		From: mail.Address{Address: "sender@example.com"},
		To:   []mail.Address{{Address: "to@example.com"}},
		Cc:   []mail.Address{{Address: "cc@example.com"}},
		Bcc:  []mail.Address{{Address: "bcc@example.com"}},
	}

	msg := sender.buildMessage(email)

	msgStr := string(msg)
	assert.Contains(t, msgStr, "To: to@example.com")
	assert.Contains(t, msgStr, "Cc: cc@example.com")
	// BCC should not be in headers
	assert.NotContains(t, msgStr, "bcc@example.com")
}

func TestSender_FormatAddress(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	addr := mail.Address{Name: "John Doe", Address: "john@example.com"}
	result := sender.formatAddress(addr)
	assert.Equal(t, "John Doe <john@example.com>", result)
}

func TestSender_FormatAddress_WithQuotes(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	addr := mail.Address{Name: `John "The Rock" Doe`, Address: "john@example.com"}
	result := sender.formatAddress(addr)
	assert.Equal(t, `John \"The Rock\" Doe <john@example.com>`, result)
}

func TestSender_FormatAddress_NoName(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	addr := mail.Address{Address: "john@example.com"}
	result := sender.formatAddress(addr)
	assert.Equal(t, "john@example.com", result)
}

func TestSender_FormatAddressList(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	addrs := []mail.Address{
		{Name: "John", Address: "john@example.com"},
		{Name: "Jane", Address: "jane@example.com"},
	}
	result := sender.formatAddressList(addrs)

	assert.Contains(t, result, "John <john@example.com>")
	assert.Contains(t, result, "Jane <jane@example.com>")
	assert.Contains(t, result, ", ")
}

func TestSender_GetEmailAddresses(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	addrs := []mail.Address{
		{Name: "John", Address: "john@example.com"},
		{Name: "Jane", Address: "jane@example.com"},
	}
	result := sender.getEmailAddresses(addrs)

	assert.Equal(t, []string{"john@example.com", "jane@example.com"}, result)
}

func TestSender_BuildMessage_WithSpecialCharactersInSubject(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Тестовое сообщение", // Cyrillic characters
		Body:    "Test",
	}

	msg := sender.buildMessage(email)
	msgStr := string(msg)

	assert.Contains(t, msgStr, "Subject: Тестовое сообщение")
	assert.True(t, strings.Contains(msgStr, "\r\n\r\nTest"))
}

func TestSender_Send_ContextCancellation(t *testing.T) {
	cfg := Config{
		Host:     "localhost",
		Port:     2525,
		Username: "test",
		Password: "test",
		TLS:      true,
	}
	sender := NewSender(cfg, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	email := mail.Email{
		From: mail.Address{Address: "sender@example.com"},
		To:   []mail.Address{{Address: "recipient@example.com"}},
	}

	err := sender.Send(ctx, email)
	// Should fail with connection error or context cancellation
	assert.Error(t, err)
}

func TestSender_Send_WithDefaultFrom(t *testing.T) {
	cfg := Config{
		Host: "localhost",
		Port: 2525,
		From: "default@example.com", // Default From address
	}
	sender := NewSender(cfg, nil)

	ctx := context.Background()
	email := mail.Email{
		// No From specified - should use default
		To: []mail.Address{{Address: "recipient@example.com"}},
	}

	err := sender.Send(ctx, email)
	// Should fail to connect, but the From address should be set correctly
	assert.Error(t, err)
	// Error should be about connection, not about missing From
	assert.NotContains(t, err.Error(), "no from address")
}

func TestSender_Send_WithDefaultFromAndEmailFrom(t *testing.T) {
	cfg := Config{
		Host: "localhost",
		Port: 2525,
		From: "default@example.com", // Default From address
	}
	sender := NewSender(cfg, nil)

	ctx := context.Background()
	email := mail.Email{
		// Email has From - should use this instead of default
		From: mail.Address{Address: "explicit@example.com"},
		To:   []mail.Address{{Address: "recipient@example.com"}},
	}

	err := sender.Send(ctx, email)
	// Should fail to connect, but the From address should be set correctly
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "no from address")
}

func TestSender_Send_MultipleEmails(t *testing.T) {
	cfg := Config{
		Host: "localhost",
		Port: 2525,
		From: "test@example.com",
	}
	sender := NewSender(cfg, nil)

	ctx := context.Background()
	emails := []mail.Email{
		{
			From:    mail.Address{Address: "sender1@example.com"},
			To:      []mail.Address{{Address: "recipient1@example.com"}},
			Subject: "Test Email 1",
			Body:    "Body 1",
		},
		{
			From:    mail.Address{Address: "sender2@example.com"},
			To:      []mail.Address{{Address: "recipient2@example.com"}},
			Subject: "Test Email 2",
			Body:    "Body 2",
		},
		{
			From:    mail.Address{Address: "sender3@example.com"},
			To:      []mail.Address{{Address: "recipient3@example.com"}},
			Subject: "Test Email 3",
			Body:    "Body 3",
		},
	}

	err := sender.Send(ctx, emails...)
	// Should fail to connect, but the loop should process all emails
	assert.Error(t, err)
}

func TestSender_Send_OnlyCcRecipients(t *testing.T) {
	cfg := Config{
		Host: "localhost",
		Port: 2525,
	}
	sender := NewSender(cfg, nil)

	ctx := context.Background()
	email := mail.Email{
		From: mail.Address{Address: "sender@example.com"},
		Cc:   []mail.Address{{Address: "cc@example.com"}},
	}

	err := sender.Send(ctx, email)
	// Should fail to connect
	assert.Error(t, err)
}

func TestSender_Send_OnlyBccRecipients(t *testing.T) {
	cfg := Config{
		Host: "localhost",
		Port: 2525,
	}
	sender := NewSender(cfg, nil)

	ctx := context.Background()
	email := mail.Email{
		From: mail.Address{Address: "sender@example.com"},
		Bcc:  []mail.Address{{Address: "bcc@example.com"}},
	}

	err := sender.Send(ctx, email)
	// Should fail to connect
	assert.Error(t, err)
}

func TestSender_BuildMessage_WithOnlyHTML(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "HTML Only Test",
		HTML:    "<h1>HTML Content</h1>",
		// Note: Body is empty, only HTML
	}

	msg := sender.buildMessage(email)

	msgStr := string(msg)
	assert.Contains(t, msgStr, "multipart/alternative")
	assert.Contains(t, msgStr, "<h1>HTML Content</h1>")
	assert.Contains(t, msgStr, "boundary_")
	// Should have both plain text (empty) and HTML parts
	assert.Contains(t, msgStr, "--boundary_")
}

func TestSender_BuildMessage_WithMultipleCustomHeaders(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Custom Headers Test",
		Body:    "Test body",
		Headers: map[string]string{
			"X-Priority":               "1 (Highest)",
			"X-MSMail-Priority":        "High",
			"X-Mailer":                 "TestMailer",
			"X-Auto-Response-Suppress": "All",
		},
	}

	msg := sender.buildMessage(email)

	msgStr := string(msg)
	assert.Contains(t, msgStr, "X-Priority: 1 (Highest)")
	assert.Contains(t, msgStr, "X-MSMail-Priority: High")
	assert.Contains(t, msgStr, "X-Mailer: TestMailer")
	assert.Contains(t, msgStr, "X-Auto-Response-Suppress: All")
}

func TestSender_BuildMessage_WithEmptySubject(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "",
		Body:    "Test body",
	}

	msg := sender.buildMessage(email)

	msgStr := string(msg)
	assert.Contains(t, msgStr, "Subject: ")
}

func TestSender_BuildMessage_VerifyStructure(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	email := mail.Email{
		From:    mail.Address{Address: "sender@example.com"},
		To:      []mail.Address{{Address: "recipient@example.com"}},
		Subject: "Structure Test",
		Body:    "Plain body",
		HTML:    "<p>HTML body</p>",
	}

	msg := sender.buildMessage(email)
	msgStr := string(msg)

	// Verify order of headers
	fromIdx := strings.Index(msgStr, "From:")
	toIdx := strings.Index(msgStr, "To:")
	subjectIdx := strings.Index(msgStr, "Subject:")
	mimeIdx := strings.Index(msgStr, "MIME-Version:")
	dateIdx := strings.Index(msgStr, "Date:")

	assert.Less(t, fromIdx, toIdx)
	assert.Less(t, toIdx, subjectIdx)
	assert.Less(t, subjectIdx, mimeIdx)
	assert.Less(t, mimeIdx, dateIdx)

	// Verify content type comes after headers
	contentTypeIdx := strings.Index(msgStr, "Content-Type:")
	assert.Greater(t, contentTypeIdx, dateIdx)
}

func TestSender_Config_WithUsernameAndPassword(t *testing.T) {
	cfg := Config{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user@example.com",
		Password: "secret",
		TLS:      true,
	}
	sender := NewSender(cfg, nil)

	assert.Equal(t, "smtp.example.com", sender.cfg.Host)
	assert.Equal(t, 587, sender.cfg.Port)
	assert.Equal(t, "user@example.com", sender.cfg.Username)
	assert.Equal(t, "secret", sender.cfg.Password)
	assert.True(t, sender.cfg.TLS)
}

func TestSender_Config_WithInsecureTLS(t *testing.T) {
	cfg := Config{
		Host:     "smtp.example.com",
		Port:     587,
		Insecure: true,
	}
	sender := NewSender(cfg, nil)

	assert.True(t, sender.cfg.Insecure)
}

func TestSender_FormatAddress_EmptyName(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	addr := mail.Address{Name: "", Address: "test@example.com"}
	result := sender.formatAddress(addr)
	assert.Equal(t, "test@example.com", result)
}

func TestSender_FormatAddressList_Empty(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	addrs := []mail.Address{}
	result := sender.formatAddressList(addrs)
	assert.Equal(t, "", result)
}

func TestSender_GetEmailAddresses_Empty(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	addrs := []mail.Address{}
	result := sender.getEmailAddresses(addrs)
	assert.Equal(t, []string{}, result)
}

func TestSender_NewSender_WithOptions(t *testing.T) {
	cfg := Config{
		Host: "localhost",
		Port: 2525,
	}
	options := &SenderOptions{}
	sender := NewSender(cfg, options)

	assert.NotNil(t, sender)
	assert.False(t, sender.closed)
}

func TestSender_NewSender_NilOptions(t *testing.T) {
	cfg := Config{
		Host: "localhost",
		Port: 2525,
	}
	sender := NewSender(cfg, nil)

	assert.NotNil(t, sender)
	assert.False(t, sender.closed)
}

func TestSender_Send_MultipleRecipients(t *testing.T) {
	cfg := Config{
		Host: "localhost",
		Port: 2525,
	}
	sender := NewSender(cfg, nil)

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
			{Address: "bcc3@example.com"},
		},
		Subject: "Multiple Recipients",
		Body:    "Test",
	}

	err := sender.Send(ctx, email)
	// Should fail to connect
	assert.Error(t, err)
}

func TestSender_Config_DefaultValues(t *testing.T) {
	cfg := Config{
		Host: "localhost",
		Port: 25,
	}
	sender := NewSender(cfg, nil)

	assert.Equal(t, "localhost", sender.cfg.Host)
	assert.Equal(t, 25, sender.cfg.Port)
	assert.False(t, sender.cfg.TLS)
	assert.False(t, sender.cfg.Insecure)
	assert.Empty(t, sender.cfg.Username)
	assert.Empty(t, sender.cfg.Password)
	assert.Empty(t, sender.cfg.From)
}

func TestSender_Config_AllFields(t *testing.T) {
	cfg := Config{
		Host:     "smtp.gmail.com",
		Port:     587,
		Username: "user@gmail.com",
		Password: "app-password",
		From:     "sender@gmail.com",
		TLS:      true,
		Insecure: false,
	}
	sender := NewSender(cfg, nil)

	assert.Equal(t, "smtp.gmail.com", sender.cfg.Host)
	assert.Equal(t, 587, sender.cfg.Port)
	assert.Equal(t, "user@gmail.com", sender.cfg.Username)
	assert.Equal(t, "app-password", sender.cfg.Password)
	assert.Equal(t, "sender@gmail.com", sender.cfg.From)
	assert.True(t, sender.cfg.TLS)
	assert.False(t, sender.cfg.Insecure)
}

func TestSender_BuildMessage_WithAllFields(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	email := mail.Email{
		From: mail.Address{
			Name:    "Sender Name",
			Address: "sender@example.com",
		},
		To: []mail.Address{
			{Name: "To 1", Address: "to1@example.com"},
			{Name: "To 2", Address: "to2@example.com"},
		},
		Cc: []mail.Address{
			{Name: "Cc 1", Address: "cc1@example.com"},
		},
		Subject: "Test with All Fields",
		Body:    "Plain text body",
		HTML:    "<html><body>HTML body</body></html>",
		Headers: map[string]string{
			"X-Custom-1": "value1",
			"X-Custom-2": "value2",
			"X-Custom-3": "value3",
		},
	}

	msg := sender.buildMessage(email)
	msgStr := string(msg)

	// Verify all fields are present
	assert.Contains(t, msgStr, "From: Sender Name <sender@example.com>")
	assert.Contains(t, msgStr, "To: To 1 <to1@example.com>")
	assert.Contains(t, msgStr, "Cc: Cc 1 <cc1@example.com>")
	assert.Contains(t, msgStr, "Subject: Test with All Fields")
	assert.Contains(t, msgStr, "X-Custom-1: value1")
	assert.Contains(t, msgStr, "X-Custom-2: value2")
	assert.Contains(t, msgStr, "X-Custom-3: value3")
	assert.Contains(t, msgStr, "Plain text body")
	assert.Contains(t, msgStr, "HTML body")
	assert.Contains(t, msgStr, "multipart/alternative")
}

func TestSender_FormatAddress_SpecialCharacters(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	tests := []struct {
		name     string
		addr     mail.Address
		expected string
	}{
		{
			name:     "simple",
			addr:     mail.Address{Address: "test@example.com"},
			expected: "test@example.com",
		},
		{
			name:     "with name",
			addr:     mail.Address{Name: "Test User", Address: "test@example.com"},
			expected: "Test User <test@example.com>",
		},
		{
			name:     "with comma in name",
			addr:     mail.Address{Name: "Last, First", Address: "test@example.com"},
			expected: "Last, First <test@example.com>",
		},
		{
			name:     "with dot in name",
			addr:     mail.Address{Name: "John Doe Jr.", Address: "test@example.com"},
			expected: "John Doe Jr. <test@example.com>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sender.formatAddress(tt.addr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSender_GetEmailAddresses_Single(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	addrs := []mail.Address{
		{Address: "single@example.com"},
	}
	result := sender.getEmailAddresses(addrs)

	assert.Equal(t, []string{"single@example.com"}, result)
}

func TestSender_GetEmailAddresses_Multiple(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	addrs := []mail.Address{
		{Name: "A", Address: "a@example.com"},
		{Name: "B", Address: "b@example.com"},
		{Name: "C", Address: "c@example.com"},
	}
	result := sender.getEmailAddresses(addrs)

	assert.Equal(t, []string{"a@example.com", "b@example.com", "c@example.com"}, result)
}

func TestSender_FormatAddressList_Single(t *testing.T) {
	cfg := Config{Host: "localhost"}
	sender := NewSender(cfg, nil)

	addrs := []mail.Address{
		{Name: "Single", Address: "single@example.com"},
	}
	result := sender.formatAddressList(addrs)

	assert.Equal(t, "Single <single@example.com>", result)
}

func TestSender_Send_ZeroEmails(t *testing.T) {
	cfg := Config{
		Host: "localhost",
		Port: 2525,
	}
	sender := NewSender(cfg, nil)

	ctx := context.Background()
	err := sender.Send(ctx)
	// Sending zero emails should succeed (does nothing)
	assert.NoError(t, err)
}

func TestSender_Send_EmptyEmails(t *testing.T) {
	cfg := Config{
		Host: "localhost",
		Port: 2525,
	}
	sender := NewSender(cfg, nil)

	ctx := context.Background()
	err := sender.Send(ctx, mail.Email{}, mail.Email{})
	// Empty emails should fail validation (no From, no To)
	assert.Error(t, err)
}
