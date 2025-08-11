package v1

type HttpServerConfig struct {
	// The address to listen on. Required.
	// Must be specified as one of the following:
	//  - "<ip>:<port>": Binds to the IP address on the given interface only
	//  - ":<port>": Binds to all interfaces
	ListenAddr string `json:"listenAddr" env:"LISTEN_ADDR"`
	// Specify TLS configuration for the HTTP server.
	// Defaults to running without TLS.
	TLS HttpTLSConfig `json:"tls"`
	// Specify authentication for the HTTP server.
	// Defaults to no authentication.
	Auth HttpAuthConfig `json:"auth,omitempty"`
}

type HttpTLSConfig struct {
	// Whether TLS is enabled for the HTTP server.
	// If enabled, certificate and key files must also be specified.
	Enabled bool `json:"enabled" env:"TLS_ENABLED"`
	// The path to the TLS certificate to serve.
	// At most one of CertFile or CertData can be specified.
	CertFile string `json:"certFile,omitempty" env:"TLS_CERT_FILE"`
	// The TLS certificate data to serve.
	// At most one of CertFile or CertData can be specified.
	CertData string `json:"certData,omitempty"`
	// The path to the TLS private key.
	KeyFile string `json:"keyFile,omitempty" env:"TLS_KEY_FILE"`
	// The TLS private key.
	// At most one of KeyFile or KeyData can be specified.
	KeyData string `json:"keyData,omitempty"`
}

type HttpAuthConfig struct {
	// Optionally specify a fixed Bearer token that can be used to authenticate incoming requests.
	AuthorizationToken string `json:"authorizationToken,omitempty" env:"AUTH_TOKEN"`
}
