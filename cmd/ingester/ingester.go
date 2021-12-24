package main

import (
	"io"
	"net/http"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/irvinlim/apple-health-ingester/pkg/backends"
	"github.com/irvinlim/apple-health-ingester/pkg/ingester"
)

type RegisterBackendFunc func(ingester *ingester.Ingester, mux *http.ServeMux) error

func RegisterBackend(backend backends.Backend, ingester *ingester.Ingester, mux *http.ServeMux, pattern string) error {
	mux.Handle(pattern, handleIngest(ingester, backend.Name()))
	if err := ingester.AddBackend(backend); err != nil {
		return err
	}

	log.WithField("backend", backend.Name()).Info("registered backend")
	return nil
}

func handleIngest(ingester *ingester.Ingester, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			_, _ = io.Copy(io.Discard, r.Body)
			_ = r.Body.Close()
		}()

		q := r.URL.Query()
		target := q.Get("target")

		if err := ingester.Ingest(r.Body, name, target); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			err := errors.Wrapf(err, "ingest error for %v", name)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}
