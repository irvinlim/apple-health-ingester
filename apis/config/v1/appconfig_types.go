package v1

// Config contains the application configuration.
//
// Configuration be loaded from configuration file, or possibly overridden from
// environment variables if the `env` struct tag is specified.
type Config struct {
	// Configuration for the ingester's HTTP server.
	HttpServer HttpServerConfig `json:"httpServer"`
	// Configuration for individual ingester backends.
	Backends BackendsConfig `json:"backends"`
}
