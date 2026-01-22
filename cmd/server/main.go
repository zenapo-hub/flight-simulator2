package main

import (
	"context"
	"flight-simulator2/internal/api"
	"flight-simulator2/internal/env"
	"flight-simulator2/internal/sim"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown on SIGINT/SIGTERM
	go func() {
		sigCh := make(chan os.Signal, 2)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	// Environment effects
	wind := env.Wind{Wx: 5.0, Wy: 2.0}
	terrain := env.Terrain{SafetyMarginM: 80.0}

	environment := env.Chain{
		Effects: []env.Environment{wind, terrain},
	}

	eng := sim.New(sim.Config{
		OriginLat:   32.0853, // pick any origin
		OriginLon:   34.7818,
		TickHz:      20,
		Environment: &environment,
	})

	go func() {
		if err := eng.Run(ctx); err != nil {
			log.Printf("engine stopped: %v", err)
		}
	}()

	httpServer := &http.Server{
		Addr:              ":8080",
		Handler:           api.NewServer(eng).Handler(),
		ReadHeaderTimeout: 3 * time.Second,
	}

	go func() {
		log.Printf("server listening on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = httpServer.Shutdown(shutdownCtx)

	log.Printf("shutdown complete")
}
