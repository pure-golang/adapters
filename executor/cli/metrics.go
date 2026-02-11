package cli

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// execDuration - гистограмма длительности выполнения команд
	execDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "executor_cli_duration_seconds",
			Help:    "Длительность выполнения CLI команд",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"command", "status"},
	)

	// execTotal - счётчик выполненных команд
	execTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "executor_cli_executions_total",
			Help: "Общее количество выполнений CLI команд",
		},
		[]string{"command", "status"},
	)
)

func init() {
	// Регистрация метрик в Prometheus
	prometheus.MustRegister(execDuration)
	prometheus.MustRegister(execTotal)
}

// recordExecution записывает метрики выполнения команды
func recordExecution(command string, status string, duration float64) {
	execDuration.WithLabelValues(command, status).Observe(duration)
	execTotal.WithLabelValues(command, status).Inc()
}
