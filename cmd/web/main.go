package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"symbol-web/internal/ai"
	"symbol-web/internal/handlers"
)

const shutdownTimeout = 10 * time.Second

func main() {
	logger := log.New(os.Stderr, "symbol-web: ", log.LstdFlags)

	root := getenv("APP_ROOT", ".")
	addr := getenv("ADDR", "127.0.0.1:8080")

	service := ai.NewService(ai.NewAIClient())

	// ctx отменяется по Ctrl+C или SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	handler := handlers.NewWeb(os.DirFS(root), service, logger)
	server := newServer(ctx, addr, handler)

	done := shutdownOnSignal(ctx, server, logger)

	logger.Printf("http://%s — режим %s", addr, serviceMode(service))

	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatal(err)
	}

	stop()
	<-done
}

// getenv возвращает значение переменной окружения или fallback, если она пустая.
func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// newServer создаёт HTTP-сервер с таймаутами и ограничением размера заголовков.
func newServer(ctx context.Context, addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: handler,

		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 * 1024,

		// Контекст каждого запроса наследуется от ctx.
		BaseContext: func(net.Listener) context.Context { return ctx },
	}
}

// shutdownOnSignal ждёт отмены ctx и аккуратно останавливает сервер.
// Возвращаемый канал закрывается, когда остановка завершена.
func shutdownOnSignal(ctx context.Context, server *http.Server, logger *log.Logger) <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Printf("завершение запросов: %v", err)
			_ = server.Close()
		}
	}()

	return done
}

func serviceMode(service *ai.Service) string {
	if service.Mock() {
		return "mock"
	}
	return "live"
}
