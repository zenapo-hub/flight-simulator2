package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"flight-simulator2/internal/api"
	"flight-simulator2/internal/env"
	"flight-simulator2/internal/sim"
)

var (
	port = flag.Int("port", 8080, "Port to listen on")
)

func main() {
	flag.Parse()

	// Origin coordinates (Tel Aviv)
	originLat := 32.0853
	originLon := 34.7818

	// Setup environment
	wind := &env.Wind{
		Wx: 5, // 5 m/s east
		Wy: 2, // 2 m/s north
	}

	terrain := &env.Terrain{
		SafetyMarginM: 10, // 10m safety margin
	}

	// Create environment chain
	envChain := &env.Chain{
		Effects: []env.Environment{wind, terrain},
	}

	// Create simulation engine
	simEngine := sim.New(sim.Config{
		OriginLat:   originLat,
		OriginLon:   originLon,
		TickHz:      20, // 20 Hz simulation
		Environment: envChain,
	})

	// Create API server
	server := api.NewServer(simEngine)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: server.Handler(),
	}

	// Start simulation engine in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := simEngine.Run(ctx); err != nil && err != context.Canceled {
			log.Printf("simulation error: %v", err)
		}
	}()

	// Start HTTP server in background
	go func() {
		log.Printf("Starting HTTP server on :%d", *port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")

	// Shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Cancel simulation context
	cancel()

	log.Println("Shutdown complete")
}
