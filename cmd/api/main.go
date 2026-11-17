package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	applicationpayment "payment-processing-platform/internal/application/payment"
	postgrespayment "payment-processing-platform/internal/infrastructure/postgres"
	httptransport "payment-processing-platform/internal/transport/http"
)

func newServer(handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func runServer(server *http.Server) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	serverStopped := make(chan struct{})

	defer signal.Stop(quit)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case <-quit:
			slog.Info("Shutdown signal received")
		case <-serverStopped:
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			slog.Error("Server shutdown failed", "error", err)
		} else {
			slog.Info("Server stopped gracefully")
		}
	}()

	err := server.ListenAndServe()

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		close(serverStopped)
		wg.Wait()
		return err
	}

	wg.Wait()
	return err
}

func run() error {
	ctx := context.Background()

	if err := godotenv.Load(); err != nil {
		slog.Info(".env file not found, using environment variables")
	}

	pool, err := postgrespayment.NewPostgresPool(ctx)
	if err != nil {
		return fmt.Errorf("initialize postgres: %w", err)
	}
	defer pool.Close()

	repository := postgrespayment.NewPaymentRepository(pool)
	service := applicationpayment.NewPaymentService(repository)
	router := httptransport.NewRouter(service)

	server := newServer(router)

	if err := runServer(server); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("run HTTP server: %w", err)
	}

	return nil
}

func main() {
	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}),
	)

	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("Application failed", "error", err)
		os.Exit(1)
	}
}
