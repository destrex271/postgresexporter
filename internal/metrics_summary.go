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
	summaryMetricTableInsertSQL = `
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
		count, sum, quantile_values, flags,
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,)
	`
)

var (
	summaryMetricTableColumns = []string{
		"count BIGINT",
		"sum   DOUBLE PRECISION",

		"quantile_values JSONB",
		"flags           INTEGER",
	}
)

type summaryMetric struct {
	resMetadata *ResourceMetadata

	summary     pmetric.Summary
	name        string
	description string
	unit        string
	metadata    pcommon.Map
}

type summaryMetricsGroup struct {
	DBType     DBType
	SchemaName string

	metrics []*summaryMetric
	count   int
}

func (g *summaryMetricsGroup) Add(resMetadata *ResourceMetadata, metric any, name, description, unit string, metadata pcommon.Map) error {
	summary, ok := metric.(pmetric.Summary)
	if !ok {
		return fmt.Errorf("metric param is not Summary type")
	}

	g.count += summary.DataPoints().Len()
	g.metrics = append(g.metrics, &summaryMetric{
		resMetadata: resMetadata,
		summary:     summary,
		name:        name,
		description: description,
		unit:        unit,
		metadata:    metadata,
	})

	return nil
}

func (g *summaryMetricsGroup) insert(ctx context.Context, client *sql.DB) error {
	logger.Debug("Inserting summary metrics")

	return fmt.Errorf("not implemented")
}

func (g *summaryMetricsGroup) createTable(ctx context.Context, client *sql.DB, metricName string) error {
	metricTableColumns := slices.Concat(getBaseMetricTableColumns(g.DBType), summaryMetricTableColumns)

	return createMetricTable(ctx, client, g.SchemaName, metricName, metricTableColumns)
}
