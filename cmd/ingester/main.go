package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	"github.com/irvinlim/apple-health-ingester/pkg/ingester"
)

func main() {
	pflag.Parse()
	mux := http.NewServeMux()

	// Add middlewares
	middlewares := []Middleware{
		createLoggingHandler(log.StandardLogger()),
		createAuthenticateHandler(),
	}
	var handler http.Handler = mux
	for _, middleware := range middlewares {
		handler = middleware(handler)
	}

	server := &http.Server{
		Addr:         listenAddr,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	// Initialize and register backends for ingester
	ingest := ingester.NewIngester()
	for _, register := range []RegisterBackendFunc{
		RegisterDebugBackend,
		RegisterInfluxDBBackend,
	} {
		if err := register(ingest, mux); err != nil {
			log.WithError(err).Fatal("add backend error")
		}
	}

	// Ensure we have at least one backend configured
	if backends := ingest.ListBackends(); len(backends) == 0 {
		log.Fatal("no backends configured, see --help")
	}

	// Start ingester
	log.Info("starting ingester")
	ingest.Start()

	// Start http server
	go func() {
		log.WithField("listen_addr", listenAddr).Info("starting http server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Panicf("cannot start http server")
		}
	}()

	// Wait for server to quit
	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	go func() {
		<-quit
		log.Info("http server shutting down")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Shut down http server with a timeout to prevent any further incoming requests.
		if err := server.Shutdown(ctx); err != nil {
			log.WithError(err).Error("could not gracefully shut down http server")
		}

		close(done)
	}()
	<-done
	log.Println("http server stopped")

	// Shutdown ingester, will block until all queues are terminated.
	log.Info("ingester shutting down")
	ingest.Shutdown()
	log.Info("ingester shut down")
}
