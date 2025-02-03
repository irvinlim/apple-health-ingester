package main

import (
	"context"
	"crypto/tls"
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

	// Set log level
	if logLevel != "" {
		level, err := log.ParseLevel(logLevel)
		if err != nil {
			log.Fatalf("cannot parse log level: %v", logLevel)
		}
		log.WithField("log_level", level).Info("setting log level")
		log.SetLevel(level)
	}

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
		Addr:              listenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			CurvePreferences: []tls.CurveID{
				tls.CurveP256,
				tls.X25519,
			},
		},
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
		var err error
		if enableTLS {
			err = server.ListenAndServeTLS(certFile, keyFile)
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			log.WithError(err).Panicf("cannot start http server")
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
