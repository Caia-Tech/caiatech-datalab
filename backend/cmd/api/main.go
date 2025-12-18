package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"caiatech-datalab/backend/internal/api"
	"caiatech-datalab/backend/internal/db"
)

func main() {
	cfg := api.LoadConfigFromEnv()

	database, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer database.Close()

	if err := db.Migrate(database, cfg.MigrationsDir); err != nil {
		log.Fatalf("db migrate: %v", err)
	}

	h := api.NewHandler(api.HandlerDeps{
		DB:         database,
		AdminToken: cfg.AdminToken,
	})

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           h.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("api listening on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Printf("api stopped")
}
