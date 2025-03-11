package internal

import (
	"context"
	"database/sql"
	"fmt"
	"slices"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

const (
	histogramMetricTableInsertSQL = `
	INSERT INTO %s.%s (
		resource_url, resource_attributes,
		scope_name, scope_version, scope_attributes, scope_dropped_attr_count, scope_url, service_name,
		name, description, unit,
		start_timestamp, timestamp,
		attribute1, attribute2, attribute3, attribute4, attribute5,
		attribute6, attribute7, attribute8, attribute9, attribute10,
		attribute11, attribute12, attribute13, attribute14, attribute15,
		attribute16, attribute17, attribute18, attribute19, attribute20,
		metadata,
		count, sum, bucket_counts, explicit_bounds, exemplars, flags, min, max, aggregation_temporality,
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,)
	`
)

var (
	histogramMetricTableColumns = []string{
		"count BIGINT",
		"sum   DOUBLE PRECISION",

		"bucket_counts   JSONB",
		"explicit_bounds JSONB",

		"exemplars JSONB",
		"flags     INTEGER",

		"min DOUBLE PRECISION",
		"max DOUBLE PRECISION",

		"aggregation_temporality VARCHAR",
	}
)

type histogramMetric struct {
	resMetadata *ResourceMetadata

	histogram   pmetric.Histogram
	name        string
	description string
	unit        string
	metadata    pcommon.Map
}

type histogramMetricsGroup struct {
	DBType     DBType
	SchemaName string

	metrics []*histogramMetric
	count   int
}

func (g *histogramMetricsGroup) Add(resMetadata *ResourceMetadata, metric any, name, description, unit string, metadata pcommon.Map) error {
	histogram, ok := metric.(pmetric.Histogram)
	if !ok {
		return fmt.Errorf("metric param is not Histogram type")
	}

	g.count += histogram.DataPoints().Len()
	g.metrics = append(g.metrics, &histogramMetric{
		resMetadata: resMetadata,
		histogram:   histogram,
		name:        name,
		description: description,
		unit:        unit,
		metadata:    metadata,
	})

	return nil
}

func (g *histogramMetricsGroup) insert(ctx context.Context, client *sql.DB) error {
	logger.Debug("Inserting histogram metrics")

	return fmt.Errorf("not implemented")
}

func (g *histogramMetricsGroup) createTable(ctx context.Context, client *sql.DB, metricName string) error {
	metricTableColumns := slices.Concat(getBaseMetricTableColumns(g.DBType), histogramMetricTableColumns)

	return createMetricTable(ctx, client, g.SchemaName, metricName, metricTableColumns)
}
