package config

import (
	"strconv"

	"k8s.io/apimachinery/pkg/util/validation/field"

	configv1 "github.com/irvinlim/apple-health-ingester/apis/config/v1"
)

// Validate performs validation on the given config.
func Validate(cfg *configv1.Config) error {
	validator := &Validator{}
	errList := validator.Validate(cfg)
	if len(errList) > 0 {
		return errList.ToAggregate()
	}
	return nil
}

// Validator is a single instance of a config validator.
type Validator struct{}

// Validate is the entrypoint to validate a config.
func (v *Validator) Validate(cfg *configv1.Config) field.ErrorList {
	fldPath := field.NewPath("config")
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, v.validateHttpServerConfig(fldPath.Child("httpServer"), cfg.HttpServer)...)
	allErrs = append(allErrs, v.validateBackendsConfig(fldPath.Child("backends"), cfg.Backends)...)
	return allErrs
}

func (v *Validator) validateHttpServerConfig(fldPath *field.Path, cfg configv1.HttpServerConfig) field.ErrorList {
	allErrs := field.ErrorList{}

	if cfg.ListenAddr == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("listenAddr"), "listenAddr is required"))
	} else if _, err := strconv.Atoi(cfg.ListenAddr); err == nil {
		// Show an error if we managed to parse it as a number, which is incorrect.
		allErrs = append(allErrs, field.Invalid(fldPath.Child("listenAddr"), cfg.ListenAddr, `should be specified as "<ip>:<port>" or ":<port>" format`))
	}

	allErrs = append(allErrs, v.validateHttpTLSConfig(fldPath.Child("tls"), cfg.TLS)...)
	return allErrs
}

func (v *Validator) validateHttpTLSConfig(fldPath *field.Path, cfg configv1.HttpTLSConfig) field.ErrorList {
	allErrs := field.ErrorList{}

	// Skip validation if not enabled.
	if !cfg.Enabled {
		return allErrs
	}

	// Exactly one of certFile / certData must be specified.
	var certSpecified int
	if cfg.CertFile != "" {
		certSpecified++
	}
	if len(cfg.CertData) > 0 {
		certSpecified++
	}
	if certSpecified == 0 {
		allErrs = append(allErrs, field.Required(fldPath, "must specify either certFile or certData"))
	} else if certSpecified > 1 {
		allErrs = append(allErrs, field.Invalid(fldPath, cfg.CertFile, "exactly one or certFile or certData must be specified"))
	}

	// Exactly one of keyFile / keyData must be specified.
	var keySpecified int
	if cfg.KeyFile != "" {
		keySpecified++
	}
	if len(cfg.KeyData) > 0 {
		keySpecified++
	}
	if keySpecified == 0 {
		allErrs = append(allErrs, field.Required(fldPath, "must specify either keyFile or keyData"))
	} else if keySpecified > 1 {
		allErrs = append(allErrs, field.Invalid(fldPath, cfg.KeyFile, "exactly one or keyFile or keyData must be specified"))
	}

	return allErrs
}

func (v *Validator) validateBackendsConfig(fldPath *field.Path, cfg configv1.BackendsConfig) field.ErrorList {
	allErrs := field.ErrorList{}

	var enabled int
	if cfg.InfluxDB.Enabled {
		enabled++
		allErrs = append(allErrs, v.validateInfluxdbBackendConfig(fldPath.Child("influxdb"), cfg.InfluxDB)...)
	}
	if cfg.LocalFile.Enabled {
		enabled++
		allErrs = append(allErrs, v.validateLocalFileBackendConfig(fldPath.Child("localFile"), cfg.LocalFile)...)
	}
	if enabled == 0 {
		allErrs = append(allErrs, field.Required(fldPath, "at least one backend must be specified"))
	}

	return allErrs
}

func (v *Validator) validateInfluxdbBackendConfig(fldPath *field.Path, cfg configv1.InfluxdbBackendConfig) field.ErrorList {
	allErrs := field.ErrorList{}

	// Skip validation if not enabled.
	if !cfg.Enabled {
		return allErrs
	}

	if cfg.ServerURL == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("serverURL"), "serverURL is required"))
	}
	if cfg.OrgName == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("orgName"), "organization name is required"))
	}
	if cfg.BucketNames.Metrics == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("bucketNames", "metrics"), "bucket name is required"))
	}
	if cfg.BucketNames.Workouts == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("bucketNames", "workouts"), "bucket name is required"))
	}

	return allErrs
}

func (v *Validator) validateLocalFileBackendConfig(fldPath *field.Path, cfg configv1.LocalFileBackendConfig) field.ErrorList {
	allErrs := field.ErrorList{}

	// Skip validation if not enabled.
	if !cfg.Enabled {
		return allErrs
	}

	if cfg.MetricsPath == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("metricsPath"), "metrics path is required"))
	}

	return allErrs
}
