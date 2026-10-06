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

func main() {
	logger := log.New(os.Stderr, "symbol-web: ", log.LstdFlags)
	root := os.Getenv("APP_ROOT")
	if root == "" {
		root = "."
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	service := ai.NewService(ai.NewAIClient())
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := &http.Server{Addr: addr, Handler: handlers.NewWeb(os.DirFS(root), service, logger), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024, BaseContext: func(net.Listener) context.Context { return ctx }}
	done := make(chan struct{})
	go func() {
		defer close(done)
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			logger.Printf("завершение запросов: %v", err)
			_ = server.Close()
		}
	}()
	mode := "live"
	if service.Mock() {
		mode = "mock"
	}
	logger.Printf("http://%s — режим %s", addr, mode)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatal(err)
	}
	stop()
	<-done
}
