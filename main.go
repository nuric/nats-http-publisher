package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Config struct {
	// Debug enables debug and pretty logging
	Debug bool `env:"DEBUG" envDefault:"false"`
	// Port at which the HTTP server will listen
	Port int `env:"PORT" envDefault:"8080"`
	// NATSURL is the URL of the NATS server
	NATSURL string `env:"NATS_URL" envDefault:"nats://localhost:4222"`
	// Name of publisher client, this is used to identify the client to NATS
	Name string `env:"NAME" envDefault:"publisher"`
}

// ---------------------------

func main() {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()
	// ---------------------------
	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	// ---------------------------
	// Setup configuration
	var conf Config
	if err := env.Parse(&conf); err != nil {
		log.Fatal().Err(err).Msg("error parsing config")
	}
	log.Info().Any("config", conf).Msg("loaded configuration")
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if conf.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	// ---------------------------
	// Setup Routes
	mux := http.NewServeMux()
	mux.Handle("/healthz", http.HandlerFunc(handlerHealthz))
	app := NewAppHandlers(conf)
	mux.Handle("/publish", http.HandlerFunc(app.handlerPublish))
	// ---------------------------
	// Add global middleware like logging, tracing, etc.
	var handler http.Handler = mux
	handler = http.TimeoutHandler(handler, 1*time.Minute, "timeout")
	handler = middlewareCORS(handler)
	handler = middlewareLogger(handler)
	handler = middlewareRecover(handler)
	// ---------------------------
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", conf.Port),
		Handler: handler,
	}
	go func() {
		log.Info().Str("addr", httpServer.Addr).Msg("listening")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("error listening and serving")
		}
	}()
	// ---------------------------
	// Graceful shutdown
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		log.Info().Msg("closing nats connections")
		app.Close()
		log.Info().Msg("closed nats connections")
	}()
	go func() {
		defer wg.Done()
		<-ctx.Done()
		shutdownCtx := context.Background()
		shutdownCtx, cancel := context.WithTimeout(shutdownCtx, 10*time.Second)
		defer cancel()
		log.Info().Msg("shutting down http server")
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "error shutting down http server: %s\n", err)
		}
		log.Info().Msg("http server shut down")
	}()
	wg.Wait()
	log.Info().Msg("exiting")
}
