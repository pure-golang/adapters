package rabbitmq

type Config struct {
	URL   string `envconfig:"RABBITMQ_URL" required:"true"` // suffix _URL from RABBITMQ_URL
	Queue string `envconfig:"RABBITMQ_DEFAULT_QUEUE"`       // suffix _DEFAULT_QUEUE from RABBITMQ_DEFAULT_QUEUE
}
