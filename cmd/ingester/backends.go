package main

import (
	"net/http"

	"github.com/pkg/errors"

	"github.com/irvinlim/apple-health-ingester/pkg/backends/influxdb"
	"github.com/irvinlim/apple-health-ingester/pkg/backends/localfile"
	"github.com/irvinlim/apple-health-ingester/pkg/ingester"
)

const (
	pathPrefix = "/api/healthautoexport/v1"
)

// RegisterDebugBackend registers the Debug backend.
func RegisterDebugBackend(ingester *ingester.Ingester, mux *http.ServeMux) error {
	if !enableLocalFile {
		return nil
	}
	backend, err := localfile.NewBackend()
	if err != nil {
		return err
	}
	return RegisterBackend(backend, ingester, mux, pathPrefix+"/localfile/ingest")
}

// RegisterInfluxDBBackend registers the InfluxDB backend.
func RegisterInfluxDBBackend(ingester *ingester.Ingester, mux *http.ServeMux) error {
	if !enableInfluxDB {
		return nil
	}
	client, err := influxdb.NewClient()
	if err != nil {
		return errors.Wrapf(err, "cannot initialize client")
	}
	backend, err := influxdb.NewBackend(client)
	if err != nil {
		return err
	}
	return RegisterBackend(backend, ingester, mux, pathPrefix+"/influxdb/ingest")
}
