package postgresexporter

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap/confmaptest"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.opentelemetry.io/collector/exporter/exporterhelper"

	"github.com/destrex271/postgresexporter/internal"
	"github.com/destrex271/postgresexporter/internal/metadata"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	cm, err := confmaptest.LoadConf(filepath.Join("testdata", "config.yaml"))
	require.NoError(t, err)

	tests := []struct {
		id       component.ID
		expected component.Config
	}{
		{
			id: component.NewIDWithName(metadata.Type, "default"),
			expected: createDefaultConfig(),
		},
		{
			id: component.NewIDWithName(metadata.Type, "postgresql"),
			expected: &Config{
				DatabaseConfig: DatabaseConfig{
					Type:     internal.DBTypePostgreSQL,
					Host:     "<host>",
					Port:     5432,
					Username: "<username>",
					Password: "<password>",
					Database: "<database>",
					Schema:   "<schema>",
					SSLmode:  "<sslmode>",
				},
				LogsTableName:   "<logs_table_name>",
				TracesTableName: "<traces_table_name>",
				CreateSchema:    false,
				TimeoutSettings: exporterhelper.NewDefaultTimeoutConfig(),
				QueueSettings:   exporterhelper.NewDefaultQueueConfig(),
			},
		},
		{
			id: component.NewIDWithName(metadata.Type, "timescaledb"),
			expected: &Config{
				DatabaseConfig: DatabaseConfig{
					Type:     internal.DBTypeTimescaleDB,
					Host:     "<host>",
					Port:     5432,
					Username: "<username>",
					Password: "<password>",
					Database: "<database>",
					Schema:   "<schema>",
					SSLmode:  "disable",
				},
				LogsTableName:   "otellogs",
				TracesTableName: "oteltraces",
				CreateSchema:    true,
				TimeoutSettings: exporterhelper.NewDefaultTimeoutConfig(),
				QueueSettings:   exporterhelper.NewDefaultQueueConfig(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.id.String(), func(t *testing.T) {
			factory := NewFactory()
			cfg := factory.CreateDefaultConfig()

			sub, err := cm.Sub(tt.id.String())
			require.NoError(t, err)
			require.NoError(t, sub.Unmarshal(cfg))

			assert.NoError(t, xconfmap.Validate(cfg))
			assert.Equal(t, tt.expected, cfg)
		})
	}
}
