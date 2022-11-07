package main

import (
	"github.com/spf13/pflag"
)

var (
	listenAddr         string
	authorizationToken string
	enableInfluxDB     bool
	enableLocalFile    bool
	enableTLS          bool
	certFile           string
	keyFile            string
)

func init() {
	pflag.StringVar(&listenAddr, "http.listenAddr", ":8080", "Address to listen on.")
	pflag.StringVar(&authorizationToken, "http.authToken", "",
		"Optional authorization token that will be used to authenticate incoming requests.")
	pflag.BoolVar(&enableInfluxDB, "backend.influxdb", false, "Enable the InfluxDB storage backend.")
	pflag.BoolVar(&enableLocalFile, "backend.localfile", false, "Enable the LocalFile storage backend.")
	pflag.BoolVar(&enableTLS, "http.enableTLS", false, "Enable TLS/HTTPS. Requires setting certificate and key files.")
	pflag.StringVar(&certFile, "http.certFile", "", "Certificate file for TLS support.")
	pflag.StringVar(&keyFile, "http.keyFile", "", "Key file for TLS support.")
}
