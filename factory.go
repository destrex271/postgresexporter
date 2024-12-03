package postgresexporter

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/postgresexporter/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

func NewFactory() exporter.Factory {
	return exporter.NewFactory(metadata.Type,
		createDefaultConfig,
		exporter.WithTraces(createTracesExporter, metadata.TracesStability),
		exporter.WithLogs(createLogsExporter, metadata.LogsStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func createLogsExporter(
	ctx context.Context,
	set exporter.Settings,
	config component.Config) (exporter.Logs, error) {

	cfg := config.(*Config)
	s, err := newTracesExporter(set.Logger, cfg)

	if err != nil {
		panic(err)
	}

	return exporterhelper.NewLogsExporter(ctx, set, cfg, s.pushLogsData)
}

func createTracesExporter(
	ctx context.Context,
	set exporter.Settings,
	config component.Config) (exporter.Logs, error) {

	cfg := config.(*Config)
	s, err := newTracesExporter(set.Logger, cfg)

	if err != nil {
		panic(err)
	}

	return exporterhelper.NewTracesExporter(ctx, set, cfg, s.pushTracesData)
}
