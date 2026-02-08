package grpc

import "io"

// Provider определяет интерфейс для gRPC сервера
type Provider interface {
	Start() error
	io.Closer
}

// Runner интерфейс для запуска в горутине
type Runner interface {
	Run()
}

// RunableProvider объединяет Provider и Runner
type RunableProvider interface {
	Provider
	Runner
}
