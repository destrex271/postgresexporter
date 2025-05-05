package internal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	conventions "go.opentelemetry.io/collector/semconv/v1.27.0"
	"go.uber.org/zap"
)

// https://developers.cloudflare.com/analytics/analytics-engine/sql-api/#table-structure
const (
	// TODO: move it to the exporter config and make configurable
	maxAttributesNumber = 20

	timestampMetricTableColumnName = "timestamp"

	timescaleDBSpecificMetricTableQuery = `
	SELECT create_hypertable('%s.%s', by_range('%s'), migrate_data => true, if_not_exists => true);
	`
)

var (
	postgreSQLNoSepMetricTableColumns = []string{
		"resource_url             VARCHAR",
		"resource_attributes      JSONB",
		"scope_name               VARCHAR",
		"scope_version            VARCHAR",
		"scope_attributes         JSONB",
		"scope_dropped_attr_count INTEGER",
		"scope_url                VARCHAR",
		"service_name             VARCHAR",

		"name        VARCHAR NOT NULL",
		"type        INTEGER",
		"description VARCHAR",
		"unit        VARCHAR",

		"start_timestamp TIMESTAMP",
		"timestamp       TIMESTAMP NOT NULL",
		"metrics    JSONB",
	}
	postgreSQLBaseMetricTableColumns = []string{
		"resource_url             VARCHAR",
		"resource_attributes      JSONB",
		"scope_name               VARCHAR",
		"scope_version            VARCHAR",
		"scope_attributes         JSONB",
		"scope_dropped_attr_count INTEGER",
		"scope_url                VARCHAR",
		"service_name             VARCHAR",

		"name        VARCHAR NOT NULL",
		"type        INTEGER",
		"description VARCHAR",
		"unit        VARCHAR",

		"start_timestamp TIMESTAMP",
		"timestamp       TIMESTAMP NOT NULL",

		"attribute1  VARCHAR",
		"attribute2  VARCHAR",
		"attribute3  VARCHAR",
		"attribute4  VARCHAR",
		"attribute5  VARCHAR",
		"attribute6  VARCHAR",
		"attribute7  VARCHAR",
		"attribute8  VARCHAR",
		"attribute9  VARCHAR",
		"attribute10 VARCHAR",
		"attribute11 VARCHAR",
		"attribute12 VARCHAR",
		"attribute13 VARCHAR",
		"attribute14 VARCHAR",
		"attribute15 VARCHAR",
		"attribute16 VARCHAR",
		"attribute17 VARCHAR",
		"attribute18 VARCHAR",
		"attribute19 VARCHAR",
		"attribute20 VARCHAR",

		"metadata JSONB",
	}

	timescaleDBBaseMetricTableColumns = []string{
		"resource_url             VARCHAR",
		"resource_attributes      JSONB",
		"scope_name               VARCHAR",
		"scope_version            VARCHAR",
		"scope_attributes         JSONB",
		"scope_dropped_attr_count INTEGER",
		"scope_url                VARCHAR",
		"service_name             VARCHAR",

		"name        VARCHAR NOT NULL",
		"type        INTEGER",
		"description VARCHAR",
		"unit        VARCHAR",

		"start_timestamp TIMESTAMPTZ",
		"timestamp       TIMESTAMPTZ NOT NULL",

		"attribute1  VARCHAR",
		"attribute2  VARCHAR",
		"attribute3  VARCHAR",
		"attribute4  VARCHAR",
		"attribute5  VARCHAR",
		"attribute6  VARCHAR",
		"attribute7  VARCHAR",
		"attribute8  VARCHAR",
		"attribute9  VARCHAR",
		"attribute10 VARCHAR",
		"attribute11 VARCHAR",
		"attribute12 VARCHAR",
		"attribute13 VARCHAR",
		"attribute14 VARCHAR",
		"attribute15 VARCHAR",
		"attribute16 VARCHAR",
		"attribute17 VARCHAR",
		"attribute18 VARCHAR",
		"attribute19 VARCHAR",
		"attribute20 VARCHAR",

		"metadata JSONB",
	}
)

// MetricsGroup is used to group metric data and insert into Postgres.
// Every type of metrics needs to implement it.
type MetricsGroup interface {
	// Units metrics data to a specific metric group
	Add(resMetadata *ResourceMetadata, metric any, name, description, unit string, metadata pcommon.Map) error

	// Creates metric table
	createTable(ctx context.Context, client *sql.DB, metricName string) error
	// Inserts metric data to db
	insert(ctx context.Context, client *sql.DB) error
	// Return metrics names
	getMetricsNames() []string
}

type ResourceMetadata struct {
	ResURL     string
	ResAttrs   pcommon.Map
	InstrScope pcommon.InstrumentationScope
	ScopeUrl   string
}

// NewMetricsModel create a model for contain different metric data
func NewMetricsGroupMap(dbtype DBType, schemaName string) map[pmetric.MetricType]MetricsGroup {
	return map[pmetric.MetricType]MetricsGroup{
		pmetric.MetricTypeGauge:                &gaugeMetricsGroup{MetricsType: pmetric.MetricTypeGauge, DBType: dbtype, SchemaName: schemaName},
		pmetric.MetricTypeSum:                  &sumMetricsGroup{MetricsType: pmetric.MetricTypeSum, DBType: dbtype, SchemaName: schemaName},
		pmetric.MetricTypeHistogram:            &histogramMetricsGroup{MetricsType: pmetric.MetricTypeHistogram, DBType: dbtype, SchemaName: schemaName},
		pmetric.MetricTypeExponentialHistogram: &expHistogramMetricsGroup{MetricsType: pmetric.MetricTypeExponentialHistogram, DBType: dbtype, SchemaName: schemaName},
		pmetric.MetricTypeSummary:              &summaryMetricsGroup{MetricsType: pmetric.MetricTypeSummary, DBType: dbtype, SchemaName: schemaName},
	}
}

// Inserts metrics data
func InsertMetrics(ctx context.Context, client *sql.DB, metricsGroupMap map[pmetric.MetricType]MetricsGroup) error {
	var errs error

	for _, m := range metricsGroupMap {
		errs = errors.Join(errs, m.insert(ctx, client))
	}

	return errs
}

func getBaseMetricTableColumns(dbtype DBType) []string {
	var tableColumns []string
	switch dbtype {
	case DBTypeTimescaleDB:
		tableColumns = timescaleDBBaseMetricTableColumns
	default:
		tableColumns = postgreSQLBaseMetricTableColumns
	}

	return tableColumns
}

func createMetricTable(ctx context.Context, client *sql.DB, schemaName, metricName string, tableColumns []string, dbtype DBType) error {
	query := fmt.Sprintf(createTableIfNotExistsSQL, schemaName, metricName, strings.Join(tableColumns, ","))
	_, err := client.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed creating schema: %w", err)
	}

	executeSpecificMetricTableQuery(ctx, client, schemaName, metricName, dbtype)

	return nil
}

// Constructs and executes a database-specific query for a given metric table.
// For example, if the dbtype is TimescaleDB it creates hypertable.
func executeSpecificMetricTableQuery(ctx context.Context, client *sql.DB, schemaName, metricName string, dbtype DBType) error {
	var specificMetricTableQuery string
	switch dbtype {
	case DBTypeTimescaleDB:
		specificMetricTableQuery = fmt.Sprintf(timescaleDBSpecificMetricTableQuery,
			schemaName, metricName, timestampMetricTableColumnName)
	default:
		specificMetricTableQuery = ""
	}

	if specificMetricTableQuery != "" {
		_, err := client.ExecContext(ctx, specificMetricTableQuery)
		if err != nil {
			logger.Warn("failed to execute specific metric table query", zap.String("dbtype", metricName), zap.Any("dbtype", dbtype))
		}
	}

	return nil
}

func getServiceName(resAttr pcommon.Map) string {
	var serviceName string
	if v, ok := resAttr.Get(conventions.AttributeServiceName); ok {
		serviceName = v.AsString()
	}

	return serviceName
}

func checkAttributesNumber(attrs pcommon.Map) error {
	if attrs.Len() > maxAttributesNumber {
		return fmt.Errorf("max attributes number exceeded")
	}

	return nil
}

func getAttributesAsSliceAndCheckIfUpdated(attrs pcommon.Map, attributesMapping *AttributesMapping) ([]*string, bool, error) {
	err := checkAttributesNumber(attrs)
	if err != nil {
		return nil, false, err
	}

	result := make([]*string, maxAttributesNumber)

	attrNameAndPosMap, err := getAttrsNameAndPosMap(attributesMapping)
	if err != nil {
		return nil, false, err
	}

	updated := false

	attrs.Range(func(k string, v pcommon.Value) bool {
		attrPos, present := attrNameAndPosMap[k]
		if !present {
			attrPos, err = findNextAvailableAttrPos(attributesMapping)
			if err != nil {
				return false
			}

			err = setAttrValueByPos(attributesMapping, attrPos, k)
			if err != nil {
				return false
			}

			updated = true
		}

		value := v.AsString()
		result[attrPos-1] = &value

		return true
	})

	if err != nil {
		return nil, false, err
	}

	return result, updated, nil
}

func getValue(intValue int64, floatValue float64, dataType any) float64 {
	switch t := dataType.(type) {
	case pmetric.NumberDataPointValueType:
		switch t {
		case pmetric.NumberDataPointValueTypeDouble:
			return floatValue
		case pmetric.NumberDataPointValueTypeInt:
			return float64(intValue)
		case pmetric.NumberDataPointValueTypeEmpty:
			return 0.0
		default:
			return 0.0
		}
	default:
		return 0.0
	}
}
