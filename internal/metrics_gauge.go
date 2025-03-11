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
	gaugeMetricTableInsertSQL = `
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
		value, exemplars, flags,
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,)
	`
)

var (
	gaugeMetricTableColumns = []string{
		"value DOUBLE PRECISION",

		"exemplars JSONB",
		"flags     INTEGER",
	}
)

type gaugeMetric struct {
	resMetadata *ResourceMetadata

	gauge       pmetric.Gauge
	name        string
	description string
	unit        string
	metadata    pcommon.Map
}

type gaugeMetricsGroup struct {
	DBType     DBType
	SchemaName string

	metrics []*gaugeMetric
	count   int
}

func (g *gaugeMetricsGroup) Add(resMetadata *ResourceMetadata, metric any, name, description, unit string, metadata pcommon.Map) error {
	gauge, ok := metric.(pmetric.Gauge)
	if !ok {
		return fmt.Errorf("metric param is not Gauge type")
	}

	g.count += gauge.DataPoints().Len()
	g.metrics = append(g.metrics, &gaugeMetric{
		resMetadata: resMetadata,
		gauge:       gauge,
		name:        name,
		description: description,
		unit:        unit,
		metadata:    metadata,
	})

	return nil
}

func (g *gaugeMetricsGroup) insert(ctx context.Context, client *sql.DB) error {
	logger.Debug("Inserting gauge metrics")

	return fmt.Errorf("not implemented")
}

func (g *gaugeMetricsGroup) createTable(ctx context.Context, client *sql.DB, metricName string) error {
	metricTableColumns := slices.Concat(getBaseMetricTableColumns(g.DBType), gaugeMetricTableColumns)

	return createMetricTable(ctx, client, g.SchemaName, metricName, metricTableColumns)
}
