package main

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"sigs.k8s.io/yaml"

	"github.com/irvinlim/apple-health-ingester/pkg/config"
	"github.com/irvinlim/apple-health-ingester/pkg/ingester"
	"github.com/irvinlim/apple-health-ingester/pkg/util/logutils"
)

var (
	defaultConfigFileLocations = []string{
		"config.yaml",         // Load from current working directory
		"config.json",         // Support both JSON and YAML (prefer to use YAML first)
		"/config/config.yaml", // Also support /config in case of container-based deployments (e.g. Docker, K8s)
		"/config/config.json", // Finally also support JSON within /config
	}
)

func main() {
	// Parse flags from command-line.
	flags, err := ParseFlags(pflag.CommandLine)
	if err != nil {
		log.Fatalf("failed to initialize command-line flags: %v", err)
	}
	mux := http.NewServeMux()

	// Set log level
	if flags.LogLevel != "" {
		level, err := log.ParseLevel(flags.LogLevel)
		if err != nil {
			log.Fatalf("cannot parse log level: %v", flags.LogLevel)
		}
		log.WithField("log_level", level).Info("setting log level")
		log.SetLevel(level)
	}

	// Load config
	cfg, err := LoadConfigAndMergeFlags(flags, config.LoadConfigArgs{
		ConfigFilePath:          flags.ConfigFile,
		OptionalConfigFilePaths: defaultConfigFileLocations,
	})
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	if log.IsLevelEnabled(log.DebugLevel) {
		cfg := cfg.DeepCopy()

		// Redact certain fields.
		if cfg.HttpServer.Auth.AuthorizationToken != "" {
			cfg.HttpServer.Auth.AuthorizationToken = "REDACTED"
		}
		if len(cfg.HttpServer.TLS.CertData) > 0 {
			cfg.HttpServer.TLS.CertData = "DATA+OMITTED"
		}
		if len(cfg.HttpServer.TLS.KeyData) > 0 {
			cfg.HttpServer.TLS.KeyData = "DATA+OMITTED"
		}
		if cfg.Backends.InfluxDB.AuthToken != "" {
			cfg.Backends.InfluxDB.AuthToken = "REDACTED"
		}

		if out, err := yaml.Marshal(cfg); err == nil {
			logutils.QuotesDisabled().WithField("config", "\n"+string(out)).Info("successfully loaded validated config")
		}
	}

	// Validate the configuration at this stage.
	if err := config.Validate(cfg); err != nil {
		log.WithError(err).Fatal("invalid config, see --help")
	}

	// Add middlewares
	middlewares := []Middleware{
		createLoggingHandler(log.StandardLogger()),
		createAuthenticateHandler(cfg),
	}
	var handler http.Handler = mux
	for _, middleware := range middlewares {
		handler = middleware(handler)
	}

	server := &http.Server{
		Addr:              cfg.HttpServer.ListenAddr,
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

	// Set up TLS.
	if tlsCfg := cfg.HttpServer.TLS; tlsCfg.Enabled {
		certData := strings.TrimSpace(tlsCfg.CertData)
		keyData := strings.TrimSpace(tlsCfg.KeyData)
		if len(certData) > 0 && len(keyData) > 0 {
			cert, err := tls.X509KeyPair([]byte(certData), []byte(keyData))
			if err != nil {
				log.Fatalf("failed to load server certificate and key: %v", err)
			}
			server.TLSConfig.Certificates = []tls.Certificate{cert}
		} else if tlsCfg.CertFile != "" && tlsCfg.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(tlsCfg.CertFile, tlsCfg.KeyFile)
			if err != nil {
				log.Fatalf("failed to load server certificate and key: %v", err)
			}
			server.TLSConfig.Certificates = []tls.Certificate{cert}
		} else {
			log.Fatalf("TLS cert and key are required, see --help")
		}
	}

	// Initialize and register backends for ingester
	ingest := ingester.NewIngester()
	for _, register := range []RegisterBackendFunc{
		RegisterDebugBackend,
		RegisterInfluxDBBackend,
	} {
		if err := register(cfg, ingest, mux); err != nil {
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
		log.WithField("listen_addr", cfg.HttpServer.ListenAddr).Info("starting http server")
		var err error
		if cfg.HttpServer.TLS.Enabled {
			// Use the cert and key from the TLSConfig.
			err = server.ListenAndServeTLS("", "")
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
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
