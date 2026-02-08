package smtp

// Config contains SMTP connection parameters.
type Config struct {
	Host     string `envconfig:"SMTP_HOST" required:"true"`     // smtp.gmail.com
	Port     int    `envconfig:"SMTP_PORT" default:"587"`       // 587 for STARTTLS, 465 for TLS
	Username string `envconfig:"SMTP_USER" required:"true"`     // username or email
	Password string `envconfig:"SMTP_PASSWORD" required:"true"` // password or app password
	From     string `envconfig:"SMTP_FROM"`                     // default from address (optional)
	TLS      bool   `envconfig:"SMTP_TLS" default:"true"`       // enable STARTTLS
	Insecure bool   `envconfig:"SMTP_INSECURE" default:"false"` // skip certificate verification
}
