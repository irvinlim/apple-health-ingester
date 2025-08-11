package main

import (
	"net/http"

	"github.com/pkg/errors"

	configv1 "github.com/irvinlim/apple-health-ingester/apis/config/v1"
	"github.com/irvinlim/apple-health-ingester/pkg/backends/influxdb"
	"github.com/irvinlim/apple-health-ingester/pkg/backends/localfile"
	"github.com/irvinlim/apple-health-ingester/pkg/ingester"
)

const (
	pathPrefix = "/api/healthautoexport/v1"
)

// RegisterDebugBackend registers the Debug backend.
func RegisterDebugBackend(cfg *configv1.Config, ingester *ingester.Ingester, mux *http.ServeMux) error {
	if cfg == nil {
		cfg = &configv1.Config{}
	}
	backendCfg := cfg.Backends.LocalFile.DeepCopy()
	if !backendCfg.Enabled {
		return nil
	}
	backend, err := localfile.NewBackend(backendCfg)
	if err != nil {
		return err
	}
	return RegisterBackend(backend, ingester, mux, pathPrefix+"/localfile/ingest")
}

// RegisterInfluxDBBackend registers the InfluxDB backend.
func RegisterInfluxDBBackend(cfg *configv1.Config, ingester *ingester.Ingester, mux *http.ServeMux) error {
	if cfg == nil {
		cfg = &configv1.Config{}
	}
	backendCfg := cfg.Backends.InfluxDB.DeepCopy()
	if !backendCfg.Enabled {
		return nil
	}
	client, err := influxdb.NewClient(backendCfg)
	if err != nil {
		return errors.Wrapf(err, "cannot initialize client")
	}
	backend, err := influxdb.NewBackend(backendCfg, client)
	if err != nil {
		return err
	}
	return RegisterBackend(backend, ingester, mux, pathPrefix+"/influxdb/ingest")
}
