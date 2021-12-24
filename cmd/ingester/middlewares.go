package main

import (
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	bearerPrefix = "Bearer "
)

type Middleware func(http.Handler) http.Handler

// createLoggingHandler returns a middleware that will log http requests.
func createLoggingHandler(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()
			defer func() {
				logger.WithFields(log.Fields{
					"method":      r.Method,
					"path":        r.URL.Path,
					"remote_addr": r.RemoteAddr,
					"user_agent":  r.UserAgent(),
					"elapsed":     time.Since(startTime),
				}).Info("http request")
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// createAuthenticateHandler returns a middleware that will authenticate
// incoming http requests.
func createAuthenticateHandler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		unauthorized := func(w http.ResponseWriter) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("unauthorized"))
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authorizationToken != "" {
				header := r.Header.Get("Authorization")

				if !strings.HasPrefix(header, bearerPrefix) {
					unauthorized(w)
					return
				}

				token := header[len(bearerPrefix):]
				if token != authorizationToken {
					unauthorized(w)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
