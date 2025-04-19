package postgresexporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/exporter/exportertest"

	"github.com/destrex271/postgresexporter/internal/metadata"
)

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, componenttest.CheckConfigStruct(cfg))
}

func TestCreateMetrics(t *testing.T) {
	cfg := createDefaultConfig()
	exporter, err := createMetricsExporter(
		context.Background(),
		exportertest.NewNopSettings(metadata.Type),
		cfg,
	)

	require.NoError(t, err)
	require.NotNil(t, exporter)
	require.NoError(t, exporter.Shutdown(context.TODO()))
}

func TestCreateLogs(t *testing.T) {
	cfg := createDefaultConfig()
	exporter, err := createLogsExporter(
		context.Background(),
		exportertest.NewNopSettings(metadata.Type),
		cfg,
	)

	require.NoError(t, err)
	require.NotNil(t, exporter)
	require.NoError(t, exporter.Shutdown(context.TODO()))
}

func TestCreateTraces(t *testing.T) {
	cfg := createDefaultConfig()
	exporter, err := createTracesExporter(
		context.Background(),
		exportertest.NewNopSettings(metadata.Type),
		cfg,
	)

	require.NoError(t, err)
	require.NotNil(t, exporter)
	require.NoError(t, exporter.Shutdown(context.TODO()))
}
