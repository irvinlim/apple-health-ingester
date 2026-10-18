package config

import (
	"github.com/caarlos0/env/v11"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"

	configv1 "github.com/irvinlim/apple-health-ingester/apis/config/v1"
)

type LoadConfigArgs struct {
	// Specify an explicit config file path to load from.
	// If specified but the file does not exist, Load will throw an error.
	ConfigFilePath string
	// List of configuration file paths to load from on a best-effort basis.
	// The first file that is found will be used.
	// Takes lower loading precedence than ConfigFilePath.
	// No error will be thrown if the file paths do not exist.
	OptionalConfigFilePaths []string
	// Optionally specify a custom environment to load from.
	// Useful for tests.
	Environment map[string]string
	// Optionally override the filesystem to load from.
	// Useful for tests.
	Fs afero.Fs
}

// LoadAndValidate is a shortcut to load and return a validated configuration.
// See Load() and Validate().
func LoadAndValidate(args LoadConfigArgs) (*configv1.Config, error) {
	cfg, err := Load(args)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load config")
	}
	if err := Validate(cfg); err != nil {
		return nil, errors.Wrapf(err, "invalid config")
	}
	return cfg, nil
}

// Load attempts to load the configuration from the following sources,
// in order of precedence from lowest to highest:
//
//  1. Default config, specified by DefaultConfig (see defaults.go)
//  2. Configuration file
//  3. Environment variables
//
// Any configuration source that is not available will be simply skipped.
//
// Note that not all configuration fields should be configurable via files or
// environment variables. Examples include log verbosity, which often must be
// specified right at the beginning of the program, or complex configuration
// types that make it cumbersome to be specified via environment variables.
func Load(args LoadConfigArgs) (*configv1.Config, error) {
	return LoadWithDefaultConfig(DefaultConfig, args)
}

// LoadWithDefaultConfig is like Load, but allows you to specify a
// different DefaultConfig.
func LoadWithDefaultConfig(defaultCfg *configv1.Config, args LoadConfigArgs) (*configv1.Config, error) {
	cfg := defaultCfg.DeepCopy()

	fs := args.Fs
	if fs == nil {
		fs = afero.NewOsFs()
	}

	// First load from configuration file.
	// Use the config file path specified explicitly if provided, otherwise use auto-discovery.
	if args.ConfigFilePath != "" {
		err := loadFromConfigFilePaths(cfg, fs, []string{args.ConfigFilePath}, true)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load config from file")
		}
	} else if len(args.OptionalConfigFilePaths) > 0 {
		err := loadFromConfigFilePaths(cfg, fs, args.OptionalConfigFilePaths, false)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load config from file")
		}
	}

	// Next, load from environment variables.
	if err := env.ParseWithOptions(cfg, env.Options{
		Environment: args.Environment,
	}); err != nil {
		return nil, errors.Wrapf(err, "failed to load config from environment")
	}

	return cfg, nil
}

func loadFromConfigFilePaths(target *configv1.Config, fs afero.Fs, paths []string, failOnError bool) error {
	// Sanity check.
	if len(paths) == 0 {
		if failOnError {
			return errors.New("empty path")
		}
		return nil
	}

	// First attempt to load the first config file that exists.
	for _, path := range paths {
		path = expandUser(path)
		exists, err := fileExists(fs, path, failOnError)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}

		log.WithField("path", path).Info("loading config from file")
		if err := unmarshalConfigFile(target, fs, path); err != nil {
			return errors.Wrapf(err, `failed to unmarshal config file at "%v"`, path)
		}

		log.WithField("path", path).Info("successfully loaded config from file")
		return nil
	}

	return nil
}

func unmarshalConfigFile(target *configv1.Config, fs afero.Fs, path string) error {
	data, err := afero.ReadFile(fs, path)
	if err != nil {
		return errors.Wrap(err, "failed to open file")
	}
	// Supports unmarshaling as either JSON or YAML.
	if err := yaml.Unmarshal(data, target); err != nil {
		log.WithField("path", path).WithError(err).Error("failed to unmarshal config file")
		return errors.Wrap(err, "failed to unmarshal config")
	}
	return nil
}
