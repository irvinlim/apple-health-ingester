package main

import (
	"github.com/spf13/pflag"
)

var (
	listenAddr         string
	authorizationToken string
	enableInfluxDB     bool
	enableLocalFile    bool
)

func init() {
	pflag.StringVar(&listenAddr, "http.listenAddr", ":8080", "Address to listen on.")
	pflag.StringVar(&authorizationToken, "http.authToken", "",
		"Optional authorization token that will be used to authenticate incoming requests.")
	pflag.BoolVar(&enableInfluxDB, "backend.influxdb", false, "Enable the InfluxDB storage backend.")
	pflag.BoolVar(&enableLocalFile, "backend.localfile", false, "Enable the LocalFile storage backend.")
}
