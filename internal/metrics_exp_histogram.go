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
	expHistogramMetricTableInsertSQL = `
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
		count, sum, scale, zero_count,
		positive_offset, positive_bucket_counts, negative_offset, negative_bucket_counts,
		exemplars, flags, min, max, zero_threshold, aggregation_temporality,
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,)
	`
)

var (
	expHistogramMetricTableColumns = []string{
		"count BIGINT",
		"sum   DOUBLE PRECISION",
		"scale INTEGER",
		"zero_count BIGINT",

		"positive_offset        INTEGER",
		"positive_bucket_counts JSONB",
		"negative_offset        INTEGER",
		"negative_bucket_counts JSONB",

		"exemplars  JSONB",
		"flags      INTEGER",

		"min            DOUBLE PRECISION",
		"max            DOUBLE PRECISION",
		"zero_threshold DOUBLE PRECISION",

		"aggregation_temporality VARCHAR",
	}
)

type expHistogramMetric struct {
	resMetadata  *ResourceMetadata

	expHistogram pmetric.ExponentialHistogram
	name         string
	description  string
	unit         string
	metadata     pcommon.Map
}

type expHistogramMetricsGroup struct {
	DBType     DBType
	SchemaName string

	metrics []*expHistogramMetric
	count   int
}

func (g *expHistogramMetricsGroup) Add(resMetadata *ResourceMetadata, metric any, name, description, unit string, metadata pcommon.Map) error {
	expHistogram, ok := metric.(pmetric.ExponentialHistogram)
	if !ok {
		return fmt.Errorf("metric param is not ExponentialHistogram type")
	}

	g.count += expHistogram.DataPoints().Len()
	g.metrics = append(g.metrics, &expHistogramMetric{
		resMetadata: resMetadata,
		expHistogram: expHistogram,
		name:         name,
		description:  description,
		unit:         unit,
		metadata:     metadata,
	})

	return nil
}

func (g *expHistogramMetricsGroup) insert(ctx context.Context, client *sql.DB) error {
	logger.Debug("Inserting exp histogram metrics")

	return fmt.Errorf("not implemented")
}

func (g *expHistogramMetricsGroup) createTable(ctx context.Context, client *sql.DB, metricName string) error {
	metricTableColumns := slices.Concat(getBaseMetricTableColumns(g.DBType), expHistogramMetricTableColumns)

	return createMetricTable(ctx, client, g.SchemaName, metricName, metricTableColumns)
}
