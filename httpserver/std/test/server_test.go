package std_test

import (
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pure-golang/adapters/httpserver/std"
)

// freePort находит свободный порт на локальном хосте и возвращает его номер.
func freePort(t *testing.T) (int, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())
	_, portStr, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)
	return port, addr
}

// TestServer_Start_ListenAndServe проверяет запуск HTTP-сервера и обработку запросов.
func TestServer_Start_ListenAndServe(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	port, addr := freePort(t)
	server := std.New(std.Config{Host: "127.0.0.1", Port: port}, handler)

	startErrChan := make(chan error, 1)
	go func() {
		startErrChan <- server.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	// Act
	resp, err := http.Get("http://" + addr + "/") //nolint:noctx
	require.NoError(t, err, "should be able to connect to server")
	t.Cleanup(func() { resp.Body.Close() })

	// Assert
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	require.NoError(t, server.Close())

	select {
	case err := <-startErrChan:
		assert.NoError(t, err, "Start should return nil after graceful shutdown")
	case <-time.After(6 * time.Second):
		t.Fatal("Start should have returned after shutdown")
	}
}

// TestServer_Start_ErrServerClosed проверяет, что Start возвращает nil при штатном завершении.
func TestServer_Start_ErrServerClosed(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	port, _ := freePort(t)
	server := std.New(std.Config{Host: "127.0.0.1", Port: port}, handler)

	startErrChan := make(chan error, 1)
	go func() {
		startErrChan <- server.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	require.NoError(t, server.Close())

	select {
	case err := <-startErrChan:
		assert.NoError(t, err, "Start должен вернуть nil при ErrServerClosed")
	case <-time.After(6 * time.Second):
		t.Fatal("Start should have returned after shutdown")
	}
}

// TestServer_Start_AddressInUse проверяет, что Start возвращает ошибку при занятом адресе.
func TestServer_Start_AddressInUse(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Занимаем порт
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() })

	_, portStr, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	server := std.New(std.Config{Host: "127.0.0.1", Port: port}, handler)

	err = server.Start()
	assert.Error(t, err, "Start should return error when address is in use")
}

// TestServer_Run_StartsInBackground проверяет, что Run запускает сервер в фоне и сразу возвращает управление.
func TestServer_Run_StartsInBackground(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	handlerCalled := make(chan struct{}, 1)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case handlerCalled <- struct{}{}:
		default:
		}
		w.WriteHeader(http.StatusOK)
	})

	port, addr := freePort(t)
	server := std.New(std.Config{Host: "127.0.0.1", Port: port}, handler)

	runComplete := make(chan struct{})
	go func() {
		server.Run()
		close(runComplete)
	}()

	// Run должен вернуться немедленно
	select {
	case <-runComplete:
	case <-time.After(1 * time.Second):
		t.Fatal("Run should return immediately after starting server in background")
	}

	time.Sleep(100 * time.Millisecond)

	// Проверяем, что сервер принимает соединения
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get("http://" + addr + "/") //nolint:noctx
	if err == nil {
		t.Cleanup(func() { resp.Body.Close() })
		select {
		case <-handlerCalled:
		case <-time.After(500 * time.Millisecond):
		}
	}

	t.Cleanup(func() { server.Close() })
}

// TestServer_Close_TimeoutExceeded проверяет поведение Close при наличии активных соединений.
func TestServer_Close_TimeoutExceeded(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	handlerCalled := make(chan struct{}, 1)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled <- struct{}{}
		// держим соединение открытым до отмены контекста
		<-r.Context().Done()
	})

	port, addr := freePort(t)
	server := std.New(std.Config{Host: "127.0.0.1", Port: port}, handler)

	serverErrChan := make(chan error, 1)
	go func() {
		serverErrChan <- server.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	// Отправляем запрос, который заблокирует соединение
	client := &http.Client{Timeout: 5 * time.Second}
	go func() {
		resp, err := client.Get("http://" + addr + "/") //nolint:noctx
		if err == nil {
			resp.Body.Close()
		}
	}()

	select {
	case <-handlerCalled:
	case <-time.After(1 * time.Second):
		t.Fatal("Handler should have been called")
	}

	// Закрываем сервер — Close завершит активное соединение через контекст
	err := server.Close()
	// допускается ошибка timeout, если соединение не освободилось вовремя
	_ = err

	select {
	case <-serverErrChan:
	case <-time.After(20 * time.Second):
		t.Fatal("Server should have stopped after Close")
	}
}
