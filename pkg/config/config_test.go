package config

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configv1 "github.com/irvinlim/apple-health-ingester/apis/config/v1"
	"github.com/irvinlim/apple-health-ingester/pkg/util/testutils"
)

func TestLoad(t *testing.T) {
	type test struct {
		name       string
		defaultCfg *configv1.Config
		files      map[string]string
		env        map[string]string
		args       LoadConfigArgs
		wantErr    assert.ErrorAssertionFunc
		want       *configv1.Config
	}
	tests := []test{
		{
			name: "error if path does not exist, use default config",
			args: LoadConfigArgs{
				ConfigFilePath: "/config.yaml",
			},
			wantErr: testutils.AssertErrorContains("file does not exist"),
		},
		{
			name: "no error if optional file does not exist, use default config",
			args: LoadConfigArgs{
				OptionalConfigFilePaths: []string{"/config.yaml"},
			},
			want: &configv1.Config{
				HttpServer: configv1.HttpServerConfig{
					ListenAddr: ":8080",
				},
			},
		},
		{
			name: "cannot load from directory",
			files: map[string]string{
				"/config/config.yaml": `
httpServer:
  listenAddr: ":5050"
`,
			},
			args: LoadConfigArgs{
				ConfigFilePath: "/config",
			},
			wantErr: testutils.AssertErrorContains("/config is a directory"),
		},
		{
			name: "skip optional file path if directory",
			files: map[string]string{
				"/config/config.yaml": `
httpServer:
  listenAddr: ":5050"
`,
			},
			args: LoadConfigArgs{
				OptionalConfigFilePaths: []string{"/config"},
			},
			want: &configv1.Config{
				HttpServer: configv1.HttpServerConfig{
					ListenAddr: ":8080",
				},
			},
		},
		{
			name: "fail on first optional config file that exists but can't be parsed",
			files: map[string]string{
				"/config.yaml": `
httpServer:
  listenAddr: ":5050
`,
				"/config2.yaml": `
httpServer:
  listenAddr: ":5051"
`,
			},
			args: LoadConfigArgs{
				OptionalConfigFilePaths: []string{"/config.yaml", "/config2.yaml"},
			},
			wantErr: testutils.AssertErrorContains("found unexpected end of stream"),
		},
		{
			name: "successfully load from config file",
			files: map[string]string{
				"/config.yaml": `
httpServer:
  listenAddr: ":5050"
`,
			},
			args: LoadConfigArgs{
				ConfigFilePath: "/config.yaml",
			},
			want: &configv1.Config{
				HttpServer: configv1.HttpServerConfig{
					ListenAddr: ":5050",
				},
			},
		},
		{
			name: "load only from non-optional config file path",
			files: map[string]string{
				"/config.yaml": `
httpServer:
  listenAddr: ":5050"
`,
				"/config2.yaml": `
httpServer:
  listenAddr: ":5051"
`,
			},
			args: LoadConfigArgs{
				ConfigFilePath:          "/config.yaml",
				OptionalConfigFilePaths: []string{"/config2.yaml"},
			},
			want: &configv1.Config{
				HttpServer: configv1.HttpServerConfig{
					ListenAddr: ":5050",
				},
			},
		},
		{
			name: "load only from first existent optional file",
			files: map[string]string{
				"/config.yaml": `
httpServer:
  listenAddr: ":5050"
`,
				"/config2.yaml": `
httpServer:
  listenAddr: ":5051"
`,
			},
			args: LoadConfigArgs{
				OptionalConfigFilePaths: []string{"/config.yaml", "/config2.yaml"},
			},
			want: &configv1.Config{
				HttpServer: configv1.HttpServerConfig{
					ListenAddr: ":5050",
				},
			},
		},
		{
			name: "override with environment variable",
			files: map[string]string{
				"/config.yaml": `
httpServer:
  listenAddr: ":5050"
`,
			},
			env: map[string]string{
				"LISTEN_ADDR": ":5051",
			},
			args: LoadConfigArgs{
				ConfigFilePath: "/config.yaml",
			},
			want: &configv1.Config{
				HttpServer: configv1.HttpServerConfig{
					ListenAddr: ":5051",
				},
			},
		},
		{
			name: "do not override with empty environment variable",
			files: map[string]string{
				"/config.yaml": `
httpServer:
  listenAddr: ":5050"
`,
			},
			env: map[string]string{
				"LISTEN_ADDR": "",
			},
			args: LoadConfigArgs{
				ConfigFilePath: "/config.yaml",
			},
			want: &configv1.Config{
				HttpServer: configv1.HttpServerConfig{
					ListenAddr: ":5050",
				},
			},
		},
		{
			name: "can load env var in nested struct",
			files: map[string]string{
				"/config.yaml": `
httpServer:
  listenAddr: ":5050"
`,
			},
			env: map[string]string{
				"TLS_ENABLED":   "true",
				"TLS_CERT_FILE": "server.crt",
				"TLS_KEY_FILE":  "server.key",
			},
			args: LoadConfigArgs{
				ConfigFilePath: "/config.yaml",
			},
			want: &configv1.Config{
				HttpServer: configv1.HttpServerConfig{
					ListenAddr: ":5050",
					TLS: configv1.HttpTLSConfig{
						Enabled:  true,
						CertFile: "server.crt",
						KeyFile:  "server.key",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			defaultCfg := tt.defaultCfg
			if defaultCfg == nil {
				defaultCfg = DefaultConfig
			}

			// Set up mock FS and environment.
			for path, data := range tt.files {
				require.NoError(t, afero.WriteFile(fs, path, []byte(data), 0644))
			}
			args := tt.args
			args.Fs = fs
			args.Environment = tt.env

			cfg, err := LoadWithDefaultConfig(defaultCfg, args)
			if testutils.WantError(t, tt.wantErr, err) {
				return
			}
			if !cmp.Equal(tt.want, cfg) {
				t.Errorf("not equal:\ndiff = %v", cmp.Diff(tt.want, cfg))
			}
		})
	}
}
