package influxdb

import (
	"crypto/tls"
	"errors"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	configv1 "github.com/irvinlim/apple-health-ingester/apis/config/v1"
)

func NewInfluxDBClient(cfg *configv1.InfluxdbBackendConfig) (influxdb2.Client, error) {
	if cfg.ServerURL == "" {
		return nil, errors.New("serverURL is not set")
	}

	options := influxdb2.DefaultOptions().
		SetTLSConfig(&tls.Config{
			InsecureSkipVerify: cfg.InsecureSkipVerify, // nolint:gosec
		})

	client := influxdb2.NewClientWithOptions(cfg.ServerURL, cfg.AuthToken, options)
	return client, nil
}
