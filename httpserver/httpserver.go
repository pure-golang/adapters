package httpserver

import "io"

type Provider interface {
	Start() error
	io.Closer
}

type Runner interface {
	Run()
}

type RunableProvider interface {
	Provider
	Runner
}
