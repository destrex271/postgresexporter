package postgresexporter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/destrex271/postgresexporter/internal"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

type metricsExporter struct {
	client *sql.DB

	config *Config
	logger *zap.Logger
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

func (e *metricsExporter) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	e.logger.Debug("Preparing to save metrics into postgres", zap.Int("Metric count", md.MetricCount()))

	metricsGroupMap := internal.NewMetricsGroupMap(e.config.DatabaseConfig.Type, e.config.DatabaseConfig.Schema)

	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		rMetrics := md.ResourceMetrics().At(i)
		resURL := rMetrics.SchemaUrl()
		resAttrs := rMetrics.Resource().Attributes()

		for j := 0; j < rMetrics.ScopeMetrics().Len(); j++ {
			metrics := rMetrics.ScopeMetrics().At(j).Metrics()
			instrScope := rMetrics.ScopeMetrics().At(j).Scope()
			scopeURL := rMetrics.ScopeMetrics().At(j).SchemaUrl()

			resMetadata := internal.ResourceMetadata{
				ResURL:     resURL,
				ResAttrs:   resAttrs,
				InstrScope: instrScope,
				ScopeUrl:   scopeURL,
			}

			for k := 0; k < metrics.Len(); k++ {
				m := metrics.At(k)

				var errs error
				switch m.Type() {
				case pmetric.MetricTypeGauge:
					errs = errors.Join(errs, metricsGroupMap[pmetric.MetricTypeGauge].Add(&resMetadata, m.Gauge(), m.Name(), m.Description(), m.Unit(), m.Metadata()))
				case pmetric.MetricTypeSum:
					errs = errors.Join(errs, metricsGroupMap[pmetric.MetricTypeSum].Add(&resMetadata, m.Sum(), m.Name(), m.Description(), m.Unit(), m.Metadata()))
				case pmetric.MetricTypeHistogram:
					errs = errors.Join(errs, metricsGroupMap[pmetric.MetricTypeHistogram].Add(&resMetadata, m.Histogram(), m.Name(), m.Description(), m.Unit(), m.Metadata()))
				case pmetric.MetricTypeExponentialHistogram:
					errs = errors.Join(errs, metricsGroupMap[pmetric.MetricTypeExponentialHistogram].Add(&resMetadata, m.ExponentialHistogram(), m.Name(), m.Description(), m.Unit(), m.Metadata()))
				case pmetric.MetricTypeSummary:
					errs = errors.Join(errs, metricsGroupMap[pmetric.MetricTypeSummary].Add(&resMetadata, m.Summary(), m.Name(), m.Description(), m.Unit(), m.Metadata()))
				case pmetric.MetricTypeEmpty:
					return fmt.Errorf("metrics type is unset")
				default:
					return fmt.Errorf("unsupported metrics type")
				}

				if errs != nil {
					e.logger.Debug(errs.Error())
					return errs
				}
			}
		}
	}

	return internal.InsertMetrics(ctx, e.client, metricsGroupMap)
}

func (e *metricsExporter) Start(ctx context.Context, host component.Host) error {
	e.logger.Debug("Starting postgres exporter")

	internal.SetLogger(e.logger)

	if e.config.SepTables {
		if !e.config.shouldCreateSchema() {
			return nil
		}

		if err := internal.CreateSchema(ctx, e.client, e.config.DatabaseConfig.Schema); err != nil {
			return err
		}

		if err := internal.CreateAttributesMappingTable(ctx, e.client, e.config.DatabaseConfig.Schema); err != nil {
			return err
		}
	} else {
		return errors.New("Separate Tables is disabled! To be implemented yet")
	}

	return nil
}

func (e *metricsExporter) Shutdown(_ context.Context) error {
	if e.client != nil {
		e.client.Close()
	}
	return nil
}
