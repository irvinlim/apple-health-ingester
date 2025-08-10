package v1

type BackendsConfig struct {
	// If specified, enables the LocalFile backend.
	LocalFile LocalFileBackendConfig `json:"localFile,omitempty"`
	// If specified, enables the InfluxDB backend.
	InfluxDB InfluxdbBackendConfig `json:"influxdb,omitempty"`
}

type LocalFileBackendConfig struct {
	// Whether the LocalFile backend is enabled.
	// Defaults to false.
	Enabled bool `json:"enabled"`
	// Output path to write metrics, with one metric per file. Required.
	// All data will be aggregated by timestamp.
	// Any existing data will be merged together.
	MetricsPath string `json:"metricsPath"`
}

type InfluxdbBackendConfig struct {
	// Whether the InfluxDB backend is enabled.
	// Defaults to false.
	Enabled bool `json:"enabled"`
	// The URL to the InfluxDB server.
	ServerURL string `json:"serverURL" env:"INFLUXDB_SERVER_URL"`
	// If true, skips TLS verification of the certificate chain and host name for
	// the InfluxDB server.
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty" env:"INFLUXDB_INSECURE_SKIP_VERIFY"`
	// Auth token to connect to InfluxDB.
	AuthToken string `json:"authToken,omitempty" env:"INFLUXDB_AUTH_TOKEN"`
	// InfluxDB organization name.
	OrgName string `json:"orgName" env:"INFLUXDB_ORG_NAME"`
	// Configuration for InfluxDB bucket names for various data points.
	BucketNames InfluxdbBucketNamesConfig `json:"bucketNames"`
	// Additional tags to add to InfluxDB for every single request, in key=value format.
	StaticTags map[string]string `json:"staticTags,omitempty"`
}

type InfluxdbBucketNamesConfig struct {
	// InfluxDB bucket name for metrics.
	Metrics string `json:"metrics" env:"INFLUXDB_METRICS_BUCKET"`
	// InfluxDB bucket name for workouts.
	Workouts string `json:"workouts" env:"INFLUXDB_WORKOUTS_BUCKET"`
}
