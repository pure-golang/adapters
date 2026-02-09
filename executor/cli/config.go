package cli

// Config содержит конфигурацию CLI executor
type Config struct {
	// Command - имя исполняемой команды (например, "ffmpeg", "gsutil", "aws")
	Command string
}
