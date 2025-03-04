package postgresexporter

import (
	"context"
	"database/sql"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

type metricsExporter struct {
	client    *sql.DB

	config    *Config
	logger    *zap.Logger
}

func newMetricsExporter(config *Config, set exporter.Settings) (*metricsExporter, error) {
	client, err := config.buildDB()
	if err != nil {
		return nil, err
	}

	return &metricsExporter{
		client: client,
		config: config,
		logger: set.Logger,
	}, nil
}

func (e *metricsExporter) ConsumeMetrics(_ context.Context, metrics pmetric.Metrics) error {
	return nil
}

func (e *metricsExporter) Start(_ context.Context, host component.Host) error {
	return nil
}

func (e *metricsExporter) Shutdown(_ context.Context) error {
	return nil
}
