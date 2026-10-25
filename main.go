package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func newServer(handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func runServer(server *http.Server) error {
	return server.ListenAndServe()
}

func main() {
	store := NewPaymentStore()

	service := NewPaymentService(store)

	mux := newRouter(service)

	server := newServer(mux)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	serverStopped := make(chan struct{})

	defer signal.Stop(quit)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case <-quit:
			log.Println("Shutdown signal received")
		case <-serverStopped:
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown failed: %v", err)
		} else {
			log.Println("Server stopped gracefully")
		}
	}()

	err := runServer(server)

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("Server error: %v", err)
		close(serverStopped)
		wg.Wait()
		return
	}

	wg.Wait()
}
