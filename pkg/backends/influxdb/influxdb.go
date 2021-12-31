package influxdb

import (
	"crypto/tls"
	"errors"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/spf13/pflag"
)

var (
	serverURL          string
	insecureSkipVerify bool
	authToken          string
	orgName            string
	metricsBucketName  string
	workoutsBucketName string
	staticTags         []string
)

func NewInfluxDBClient() (influxdb2.Client, error) {
	if serverURL == "" {
		return nil, errors.New("--influxdb.serverURL is not set")
	}

	options := influxdb2.DefaultOptions().
		SetTLSConfig(&tls.Config{
			InsecureSkipVerify: insecureSkipVerify,
		})

	client := influxdb2.NewClientWithOptions(serverURL, authToken, options)
	return client, nil
}

func init() {
	pflag.StringVar(&serverURL, "influxdb.serverURL", "", "Server URL for InfluxDB.")
	pflag.BoolVar(&insecureSkipVerify, "influxdb.insecureSkipVerify", false,
		"Skip TLS verification of the certificate chain and host name for the InfluxDB server.")
	pflag.StringVar(&authToken, "influxdb.authToken", "", "Auth token to connect to InfluxDB.")
	pflag.StringVar(&orgName, "influxdb.orgName", "", "InfluxDB organization name.")
	pflag.StringVar(&metricsBucketName, "influxdb.metricsBucketName", "", "InfluxDB bucket name for metrics.")
	pflag.StringVar(&workoutsBucketName, "influxdb.workoutsBucketName", "", "InfluxDB bucket name for workouts.")
	pflag.StringSliceVar(&staticTags, "influxdb.staticTags", nil,
		"Additional tags to add to InfluxDB for every single request, in key=value format.")
}
