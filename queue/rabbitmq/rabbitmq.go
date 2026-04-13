package rabbitmq

type Config struct {
	URL string `envconfig:"RABBITMQ_URL" required:"true"` // suffix _URL from RABBITMQ_URL
}
