package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gopherpay/internal/audit"
	"gopherpay/internal/billing"
	"gopherpay/internal/config"
	apphttp "gopherpay/internal/http"
	"gopherpay/internal/middleware"
	"gopherpay/internal/worker"
	"gopherpay/pkg/logger"
)

func main() {

	// cfg := config.LoadDBConfig()

	// if err != nil {
	// 	log.Fatal(err)
	// }
	cfg, err := config.LoadDBConfig()
	db, err := config.ConnectDB(cfg)
	if err != nil {
		log.Fatal(err)
	}

	logr := logger.NewLogger()

	repo := billing.NewMySQLRepository(db)
	auditRepo := audit.NewMySQLRepository(db)
	service := billing.NewService(repo, auditRepo, logr)

	pool := worker.NewPool(100, service, logr) //lower buffer size to test backpressure (429)
	pool.Start(10)

	// handler := apphttp.NewTransferHandler(pool)
	handler := apphttp.NewTransferHandler(pool, auditRepo)
	healthHandler := apphttp.NewHealthHandler(db)

	mux := http.NewServeMux()
	mux.Handle("/transfer", middleware.RequestID(handler))
	mux.Handle("/health", healthHandler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		log.Println("Server running on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	server.Shutdown(ctx)
	pool.Shutdown()

	log.Println("Server stopped gracefully")
}
