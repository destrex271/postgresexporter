package postgresexporter

import (
	"context"
	"fmt"
	"log"

	"github.com/destrex271/postgresexporter/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

func NewFactory() exporter.Factory {
	return exporter.NewFactory(metadata.Type,
		createDefaultConfig,
		exporter.WithLogs(createLogsExporter, metadata.LogsStability),
		exporter.WithTraces(createTracesExporter, metadata.TracesStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		DatabaseConfig: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Username: "postgres",
			Password: "postgres",
			Database: "postgres",
			SSLmode:  "disabled",
		},
		LogsTableName:    "otellogs",
		TracesTableName:  "oteltraces",
		MetricsTableName: "otelmetrics",
		TimeoutSettings: exporterhelper.NewDefaultTimeoutConfig(),
		QueueSettings: exporterhelper.NewDefaultQueueConfig(),
	}
}

func createLogsExporter(
	ctx context.Context,
	set exporter.Settings,
	config component.Config,
) (exporter.Logs, error) {
	cfg := config.(*Config)
	log.Println("CREATING LOGS EXPORTER")
	s, err := newLogsExporter(set.Logger, cfg)
	if err != nil {
		panic(err)
	}

	return exporterhelper.NewLogs(
		ctx,
		set,
		cfg,
		s.pushLogsData,
		exporterhelper.WithStart(s.start),
		exporterhelper.WithShutdown(s.shutdown),
	)
}

func createTracesExporter(
	ctx context.Context,
	set exporter.Settings,
	config component.Config,
) (exporter.Traces, error) {
	log.Println("CREATING TRACES EXPORTER")
	cfg := config.(*Config)
	s, err := newTracesExporter(set.Logger, cfg)
	if err != nil {
		panic(err)
	}

	return exporterhelper.NewTraces(
		ctx,
		set,
		cfg,
		s.pushTraceData,
		exporterhelper.WithStart(s.start),
		exporterhelper.WithShutdown(s.shutdown),
	)
}
