package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	configv1 "github.com/irvinlim/apple-health-ingester/apis/config/v1"
	"github.com/irvinlim/apple-health-ingester/pkg/config"
)

// Flags is the root data structure for defining command-line flags for the application.
//
// Currently, we support quite a fair bit of business logic flags which are also available to be specified via config.
// Any new flags that control business logic should not be added here, but instead should be added to the configuration schema.
// Using configuration files should be the preferred way moving forward.
// Existing flags will be retained for backwards compatibility for now.
type Flags struct {
	// The configured log level.
	LogLevel string
	// The configuration file to load.
	ConfigFile string
	// Flags for the HTTP server.
	HttpServer HttpServerFlags
	// Flags for ingester backends.
	Backends BackendFlags
}

// BindFlagSet registers the flags into the given FlagSet.
func (f *Flags) BindFlagSet(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.LogLevel, "log", "info", "Log level to use.")
	flagSet.StringVarP(&f.ConfigFile, "config", "c", "",
		`Path to YAML or JSON configuration file.
However, config specified via environment variables or command-line arguments will still take greater precedence.
If not specified, attempts to discover configuration files at the following locations in order:
  `+strings.Join(defaultConfigFileLocations, ", "))

	f.HttpServer.BindFlagSet(flagSet)
	f.Backends.BindFlagSet(flagSet)
}

// Merge down the flags to override the config.
func (f *Flags) Merge(cfg *configv1.Config) error {
	if err := f.HttpServer.Merge(&cfg.HttpServer); err != nil {
		return errors.Wrap(err, "failed to merge HttpServerFlags")
	}
	if err := f.Backends.Merge(&cfg.Backends); err != nil {
		return errors.Wrapf(err, "failed to merge BackendFlags")
	}
	return nil
}

type HttpServerFlags struct {
	// The address to listen on.
	ListenAddr string
	// Optional Bearer token to authorize incoming requests.
	AuthorizationToken string
	// Optionally enable TLS.
	EnableTLS bool
	// The TLS certificate to serve.
	CertFile string
	// The TLS private key.
	KeyFile string
}

// BindFlagSet registers the flags into the given FlagSet.
func (f *HttpServerFlags) BindFlagSet(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&f.ListenAddr, "http.listenAddr", "", `Address to listen on.
Environment variable: $LISTEN_ADDR`)
	flagSet.StringVar(&f.AuthorizationToken, "http.authToken", "",
		`Optional authorization token that will be used to authenticate incoming requests.
Environment variable: $AUTH_TOKEN`)
	flagSet.BoolVar(&f.EnableTLS, "http.enableTLS", false, `Enable TLS/HTTPS. Requires setting certificate and key files.
Environment variable: $TLS_ENABLED`)
	flagSet.StringVar(&f.CertFile, "http.certFile", "", `Certificate file for TLS support.
Environment variable: $TLS_CERT_FILE`)
	flagSet.StringVar(&f.KeyFile, "http.keyFile", "", `Key file for TLS support.
Environment variable: $TLS_KEY_FILE`)
}

// Merge down the flags to override the config.
func (f *HttpServerFlags) Merge(cfg *configv1.HttpServerConfig) error {
	if f.ListenAddr != "" {
		cfg.ListenAddr = f.ListenAddr
	}
	if f.AuthorizationToken != "" {
		cfg.Auth.AuthorizationToken = f.AuthorizationToken
	}
	if f.EnableTLS {
		cfg.TLS.Enabled = f.EnableTLS
	}
	if f.CertFile != "" {
		cfg.TLS.CertFile = f.CertFile
	}
	if f.KeyFile != "" {
		cfg.TLS.KeyFile = f.KeyFile
	}
	return nil
}

type BackendFlags struct {
	// Flags for InfluxDB backend.
	InfluxDB InfluxdbBackendFlags
	// Flags for LocalFile backend.
	LocalFile LocalFileBackendFlags
}

// BindFlagSet registers the flags into the given FlagSet.
func (f *BackendFlags) BindFlagSet(flagSet *pflag.FlagSet) {
	f.InfluxDB.BindFlagSet(flagSet)
	f.LocalFile.BindFlagSet(flagSet)
}

// Merge down the flags to override the config.
func (f *BackendFlags) Merge(cfg *configv1.BackendsConfig) error {
	if err := f.InfluxDB.Merge(&cfg.InfluxDB); err != nil {
		return errors.Wrap(err, "failed to merge InfluxdbBackendFlags")
	}
	if err := f.LocalFile.Merge(&cfg.LocalFile); err != nil {
		return errors.Wrap(err, "failed to merge LocalFileBackendFlags")
	}
	return nil
}

type InfluxdbBackendFlags struct {
	// Whether the backend is enabled.
	Enabled bool
	// The InfluxDB server URL.
	ServerURL string
	// Whether to skip TLS verification of the certificate chain and host name for the InfluxDB server.
	InsecureSkipVerify bool
	// Auth token to connect to InfluxDB.
	AuthToken string
	// InfluxDB organization name.
	OrgName string
	// InfluxDB bucket name for metrics.
	MetricsBucketName string
	// InfluxDB bucket name for workouts.
	WorkoutsBucketName string
	// Additional tags to add to InfluxDB for every single request, in key=value format.
	StaticTags []string
}

// BindFlagSet registers the flags into the given FlagSet.
func (f *InfluxdbBackendFlags) BindFlagSet(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(&f.Enabled, "backend.influxdb", false, "Enable the InfluxDB storage backend.")
	flagSet.StringVar(&f.ServerURL, "influxdb.serverURL", "", `Server URL for InfluxDB.
Environment variable: $INFLUXDB_SERVER_URL`)
	flagSet.BoolVar(&f.InsecureSkipVerify, "influxdb.insecureSkipVerify", false,
		`Skip TLS verification of the certificate chain and host name for the InfluxDB server.
Environment variable: $INFLUXDB_INSECURE_SKIP_VERIFY`)
	flagSet.StringVar(&f.AuthToken, "influxdb.authToken", "", `Auth token to connect to InfluxDB.
Environment variable: $INFLUXDB_AUTH_TOKEN`)
	flagSet.StringVar(&f.OrgName, "influxdb.orgName", "", `InfluxDB organization name.
Environment variable: $INFLUXDB_ORG_NAME`)
	flagSet.StringVar(&f.MetricsBucketName, "influxdb.metricsBucketName", "", `InfluxDB bucket name for metrics.
Environment variable: $INFLUXDB_METRICS_BUCKET`)
	flagSet.StringVar(&f.WorkoutsBucketName, "influxdb.workoutsBucketName", "", `InfluxDB bucket name for workouts.
Environment variable: $INFLUXDB_WORKOUTS_BUCKET`)
	flagSet.StringSliceVar(&f.StaticTags, "influxdb.staticTags", nil,
		"Additional tags to add to InfluxDB for every single request, in key=value format.")
}

// Merge down the flags to override the config.
func (f *InfluxdbBackendFlags) Merge(cfg *configv1.InfluxdbBackendConfig) error {
	if f.Enabled {
		cfg.Enabled = f.Enabled
	}
	if f.ServerURL != "" {
		cfg.ServerURL = f.ServerURL
	}
	if f.InsecureSkipVerify {
		cfg.InsecureSkipVerify = true
	}
	if f.AuthToken != "" {
		cfg.AuthToken = f.AuthToken
	}
	if f.OrgName != "" {
		cfg.OrgName = f.OrgName
	}
	if f.MetricsBucketName != "" {
		cfg.BucketNames.Metrics = f.MetricsBucketName
	}
	if f.WorkoutsBucketName != "" {
		cfg.BucketNames.Workouts = f.WorkoutsBucketName
	}
	if len(f.StaticTags) > 0 {
		// Throw an error if both command-line args and config file args are specified.
		if len(cfg.StaticTags) > 0 {
			return errors.New("cannot specify static tags via both --influxdb.staticTags and from config file")
		}
		if cfg.StaticTags == nil {
			cfg.StaticTags = make(map[string]string, len(f.StaticTags))
		}

		for _, tag := range f.StaticTags {
			tokens := strings.SplitN(tag, "=", 2)
			if len(tokens) != 2 {
				return fmt.Errorf("invalid static tag %v", tag)
			}
			cfg.StaticTags[tokens[0]] = tokens[1]
		}
	}
	return nil
}

type LocalFileBackendFlags struct {
	// Whether the backend is enabled.
	Enabled bool
	// Output path to write metrics
	MetricsPath string
}

// BindFlagSet registers the flags into the given FlagSet.
func (f *LocalFileBackendFlags) BindFlagSet(flagSet *pflag.FlagSet) {
	flagSet.BoolVar(&f.Enabled, "backend.localfile", false, "Enable the LocalFile storage backend.")
	flagSet.StringVar(&f.MetricsPath, "localfile.metricsPath", "",
		"Output path to write metrics, with one metric per file. All data will be aggregated by timestamp. "+
			"Any existing data will be merged together.")
}

// Merge down the flags to override the config.
func (f *LocalFileBackendFlags) Merge(cfg *configv1.LocalFileBackendConfig) error {
	if f.MetricsPath != "" {
		cfg.MetricsPath = f.MetricsPath
	}
	return nil
}

// ParseFlags will initialize and parse Flags.
func ParseFlags(flagSet *pflag.FlagSet) (*Flags, error) {
	flags := &Flags{}

	// Recursively bind the flagSet.
	flags.BindFlagSet(flagSet)

	// Parse the arguments.
	if err := flagSet.Parse(os.Args[1:]); err != nil {
		return nil, errors.Wrapf(err, "failed to parse command line")
	}

	return flags, nil
}

// LoadConfigAndMergeFlags is the main entrypoint to load configuration from all
// configuration sources, while also merging down any command-line flags that
// can override the
//
// Specifically, configuration will be loaded from the following sources, in
// order of precedence from lowest to highest:
//
//  1. Default config, specified by DefaultConfig (see defaults.go)
//  2. Configuration file
//  3. Environment variables
//  4. Command-line arguments
func LoadConfigAndMergeFlags(flags *Flags, args config.LoadConfigArgs) (*configv1.Config, error) {
	// First load the configuration from file.
	// We don't need to validate the config yet as we'll override with command-line args later.
	cfg, err := config.Load(args)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot load config")
	}

	// Override the configuration with flags if specified.
	cfg = cfg.DeepCopy()
	if err := flags.Merge(cfg); err != nil {
		return nil, errors.Wrapf(err, "cannot merge flags with command-line arguments")
	}

	// Now validate the config since everything is populated.
	if err := config.Validate(cfg); err != nil {
		log.WithError(err).Error("invalid config, see --help")
		return nil, errors.Wrapf(err, "invalid config")
	}

	return cfg, nil
}
