package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edgehub/edgehub/internal/config"
	"github.com/edgehub/edgehub/internal/k8s"
	"github.com/edgehub/edgehub/internal/scheduler"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	k8sClient, err := k8s.NewClientset(ctx, cfg.Kubernetes)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	sched := scheduler.NewScheduler(cfg.Scheduler, k8sClient)

	if err := sched.Start(ctx); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    ":9090",
		Handler: mux,
	}

	go func() {
		log.Println("Starting scheduler metrics server on :9090")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down scheduler...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Scheduler exited properly")
}
