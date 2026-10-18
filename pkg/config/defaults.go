package config

import (
	configv1 "github.com/irvinlim/apple-health-ingester/apis/config/v1"
)

var (
	DefaultConfig = &configv1.Config{
		HttpServer: configv1.HttpServerConfig{
			ListenAddr: ":8080",
		},
	}
)
