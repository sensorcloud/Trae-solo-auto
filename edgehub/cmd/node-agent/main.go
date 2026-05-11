package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edgehub/edgehub/internal/agent"
	"github.com/edgehub/edgehub/internal/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg := loadConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodeAgent := agent.NewNodeAgent(cfg)

	if err := nodeAgent.Start(ctx); err != nil {
		log.Fatalf("Failed to start node agent: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","agent":"edgehub-node-agent"}`))
	})
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    ":9100",
		Handler: mux,
	}

	go func() {
		log.Println("Starting node agent metrics server on :9100")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down node agent...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	if err := nodeAgent.Stop(); err != nil {
		log.Printf("Error stopping node agent: %v", err)
	}

	log.Println("Node agent exited properly")
}

func loadConfig() *config.Config {
	return config.Load()
}
