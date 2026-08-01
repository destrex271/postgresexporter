package internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/destrex271/postgresexporter/internal/db"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

const (
	genericMetricTableInsertSQL = `
	INSERT INTO "%s"."%s" (
		resource_url, resource_attributes,
		scope_name, scope_version, scope_attributes, scope_dropped_attr_count, scope_url, service_name,
		name, type, description, unit,
		start_timestamp, timestamp,
		metrics
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
	`
)

type genericMetric struct {
	resMetadata *ResourceMetadata

	metric   pmetric.Metric
	metadata pcommon.Map
}

type GenericMetricsGroup struct {
	DBType     DBType
	SchemaName string
	TableName  string

	metrics []*genericMetric
	count   int
}

func NewGenericMetricsGroup(dbtype DBType, schemaName, tableName string) *GenericMetricsGroup {
	return &GenericMetricsGroup{
		DBType:     dbtype,
		SchemaName: schemaName,
		TableName:  tableName,
	}
}

func (g *GenericMetricsGroup) Add(resMetadata *ResourceMetadata, metric pmetric.Metric) error {
	g.count += metricDataPointCount(metric)
	g.metrics = append(g.metrics, &genericMetric{
		resMetadata: resMetadata,
		metric:      metric,
		metadata:    metric.Metadata(),
	})

	return nil
}

func metricDataPointCount(metric pmetric.Metric) int {
	switch metric.Type() {
	case pmetric.MetricTypeGauge:
		return metric.Gauge().DataPoints().Len()
	case pmetric.MetricTypeSum:
		return metric.Sum().DataPoints().Len()
	case pmetric.MetricTypeHistogram:
		return metric.Histogram().DataPoints().Len()
	case pmetric.MetricTypeExponentialHistogram:
		return metric.ExponentialHistogram().DataPoints().Len()
	case pmetric.MetricTypeSummary:
		return metric.Summary().DataPoints().Len()
	default:
		return 0
	}
}

func (g *GenericMetricsGroup) CreateTable(ctx context.Context, client *sql.DB) error {
	tableColumns := getNoSepMetricTableColumns(g.DBType)

	return createMetricTable(ctx, client, g.SchemaName, g.TableName, tableColumns, g.DBType)
}

func (g *GenericMetricsGroup) Insert(ctx context.Context, client *sql.DB) error {
	logger.Debug("Inserting generic metrics")

	if g.count == 0 {
		return nil
	}

	exists, err := CheckIfTableExists(ctx, client, g.SchemaName, g.TableName)
	if err != nil {
		return err
	}

	if !exists {
		err = g.CreateTable(ctx, client)
		if err != nil {
			return err
		}
	}

	var errs error
	for _, m := range g.metrics {
		err := db.DoWithTx(ctx, client, func(tx *sql.Tx) error {
			statement, err := tx.PrepareContext(ctx, fmt.Sprintf(genericMetricTableInsertSQL, g.SchemaName, g.TableName))
			if err != nil {
				return err
			}

			defer func() {
				_ = statement.Close()
			}()

			resAttrs, scopeAttrs, metadata, err := marshalMetricMetadata(m)
			if err != nil {
				return err
			}

			switch m.metric.Type() {
			case pmetric.MetricTypeGauge:
				errs = errors.Join(errs, g.insertGauge(ctx, statement, m, resAttrs, scopeAttrs, metadata))
			case pmetric.MetricTypeSum:
				errs = errors.Join(errs, g.insertSum(ctx, statement, m, resAttrs, scopeAttrs, metadata))
			case pmetric.MetricTypeHistogram:
				errs = errors.Join(errs, g.insertHistogram(ctx, statement, m, resAttrs, scopeAttrs, metadata))
			case pmetric.MetricTypeExponentialHistogram:
				errs = errors.Join(errs, g.insertExponentialHistogram(ctx, statement, m, resAttrs, scopeAttrs, metadata))
			case pmetric.MetricTypeSummary:
				errs = errors.Join(errs, g.insertSummary(ctx, statement, m, resAttrs, scopeAttrs, metadata))
			}

			return nil
		})
		errs = errors.Join(errs, err)
	}
	if errs != nil {
		return fmt.Errorf("insert generic metrics failed: %w", errs)
	}

	return nil
}

func (g *GenericMetricsGroup) insertGauge(ctx context.Context, statement *sql.Stmt, m *genericMetric, resAttrs, scopeAttrs, metadata []byte) error {
	var errs error

	for i := range m.metric.Gauge().DataPoints().Len() {
		dp := m.metric.Gauge().DataPoints().At(i)

		if dp.Timestamp().AsTime().IsZero() {
			errs = errors.Join(errs, fmt.Errorf("data points with the 0 value for TimeUnixNano SHOULD be rejected by consumers"))
			continue
		}

		metricsJSON, err := json.Marshal(map[string]any{
			"attributes": dp.Attributes().AsRaw(),
			"value":      getValue(dp.IntValue(), dp.DoubleValue(), dp.ValueType()),
			"exemplars":  exemplarsToRaw(dp.Exemplars()),
			"flags":      uint32(dp.Flags()),
		})
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}

		err = g.insertDataPoint(ctx, statement, m, resAttrs, scopeAttrs, metadata, dp.StartTimestamp().AsTime(), dp.Timestamp().AsTime(), metricsJSON)
		if err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

func (g *GenericMetricsGroup) insertSum(ctx context.Context, statement *sql.Stmt, m *genericMetric, resAttrs, scopeAttrs, metadata []byte) error {
	var errs error

	for i := range m.metric.Sum().DataPoints().Len() {
		dp := m.metric.Sum().DataPoints().At(i)

		if dp.Timestamp().AsTime().IsZero() {
			errs = errors.Join(errs, fmt.Errorf("data points with the 0 value for TimeUnixNano SHOULD be rejected by consumers"))
			continue
		}

		metricsJSON, err := json.Marshal(map[string]any{
			"attributes":              dp.Attributes().AsRaw(),
			"value":                   getValue(dp.IntValue(), dp.DoubleValue(), dp.ValueType()),
			"exemplars":               exemplarsToRaw(dp.Exemplars()),
			"flags":                   uint32(dp.Flags()),
			"aggregation_temporality": int32(m.metric.Sum().AggregationTemporality()),
			"is_monotonic":            m.metric.Sum().IsMonotonic(),
		})
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}

		err = g.insertDataPoint(ctx, statement, m, resAttrs, scopeAttrs, metadata, dp.StartTimestamp().AsTime(), dp.Timestamp().AsTime(), metricsJSON)
		if err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

func (g *GenericMetricsGroup) insertHistogram(ctx context.Context, statement *sql.Stmt, m *genericMetric, resAttrs, scopeAttrs, metadata []byte) error {
	var errs error

	for i := range m.metric.Histogram().DataPoints().Len() {
		dp := m.metric.Histogram().DataPoints().At(i)

		if dp.Timestamp().AsTime().IsZero() {
			errs = errors.Join(errs, fmt.Errorf("data points with the 0 value for TimeUnixNano SHOULD be rejected by consumers"))
			continue
		}

		metricsJSON, err := json.Marshal(map[string]any{
			"attributes":              dp.Attributes().AsRaw(),
			"count":                   dp.Count(),
			"sum":                     dp.Sum(),
			"bucket_counts":           dp.BucketCounts().AsRaw(),
			"explicit_bounds":         dp.ExplicitBounds().AsRaw(),
			"exemplars":               exemplarsToRaw(dp.Exemplars()),
			"flags":                   uint32(dp.Flags()),
			"min":                     dp.Min(),
			"max":                     dp.Max(),
			"aggregation_temporality": int32(m.metric.Histogram().AggregationTemporality()),
		})
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}

		err = g.insertDataPoint(ctx, statement, m, resAttrs, scopeAttrs, metadata, dp.StartTimestamp().AsTime(), dp.Timestamp().AsTime(), metricsJSON)
		if err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

func (g *GenericMetricsGroup) insertExponentialHistogram(ctx context.Context, statement *sql.Stmt, m *genericMetric, resAttrs, scopeAttrs, metadata []byte) error {
	var errs error

	for i := range m.metric.ExponentialHistogram().DataPoints().Len() {
		dp := m.metric.ExponentialHistogram().DataPoints().At(i)

		if dp.Timestamp().AsTime().IsZero() {
			errs = errors.Join(errs, fmt.Errorf("data points with the 0 value for TimeUnixNano SHOULD be rejected by consumers"))
			continue
		}

		metricsJSON, err := json.Marshal(map[string]any{
			"attributes":              dp.Attributes().AsRaw(),
			"count":                   dp.Count(),
			"sum":                     dp.Sum(),
			"scale":                   dp.Scale(),
			"zero_count":              dp.ZeroCount(),
			"positive_offset":         dp.Positive().Offset(),
			"positive_bucket_counts":  dp.Positive().BucketCounts().AsRaw(),
			"negative_offset":         dp.Negative().Offset(),
			"negative_bucket_counts":  dp.Negative().BucketCounts().AsRaw(),
			"exemplars":               exemplarsToRaw(dp.Exemplars()),
			"flags":                   uint32(dp.Flags()),
			"min":                     dp.Min(),
			"max":                     dp.Max(),
			"zero_threshold":          dp.ZeroThreshold(),
			"aggregation_temporality": int32(m.metric.ExponentialHistogram().AggregationTemporality()),
		})
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}

		err = g.insertDataPoint(ctx, statement, m, resAttrs, scopeAttrs, metadata, dp.StartTimestamp().AsTime(), dp.Timestamp().AsTime(), metricsJSON)
		if err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

func (g *GenericMetricsGroup) insertSummary(ctx context.Context, statement *sql.Stmt, m *genericMetric, resAttrs, scopeAttrs, metadata []byte) error {
	var errs error

	for i := range m.metric.Summary().DataPoints().Len() {
		dp := m.metric.Summary().DataPoints().At(i)

		if dp.Timestamp().AsTime().IsZero() {
			errs = errors.Join(errs, fmt.Errorf("data points with the 0 value for TimeUnixNano SHOULD be rejected by consumers"))
			continue
		}

		metricsJSON, err := json.Marshal(map[string]any{
			"attributes":      dp.Attributes().AsRaw(),
			"count":           dp.Count(),
			"sum":             dp.Sum(),
			"quantile_values": quantileValuesToRaw(dp.QuantileValues()),
			"flags":           uint32(dp.Flags()),
		})
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}

		err = g.insertDataPoint(ctx, statement, m, resAttrs, scopeAttrs, metadata, dp.StartTimestamp().AsTime(), dp.Timestamp().AsTime(), metricsJSON)
		if err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

func (g *GenericMetricsGroup) insertDataPoint(ctx context.Context, statement *sql.Stmt, m *genericMetric, resAttrs, scopeAttrs, metadata []byte, startTimestamp, timestamp time.Time, metricsJSON []byte) error {
	_, err := statement.ExecContext(ctx,
		m.resMetadata.ResURL, resAttrs,
		m.resMetadata.InstrScope.Name(),
		m.resMetadata.InstrScope.Version(),
		scopeAttrs,
		m.resMetadata.InstrScope.DroppedAttributesCount(),
		m.resMetadata.ScopeUrl,
		getServiceName(m.resMetadata.ResAttrs),
		m.metric.Name(), int32(m.metric.Type()), m.metric.Description(), m.metric.Unit(),
		startTimestamp, timestamp,
		metricsJSON,
	)

	return err
}

func marshalMetricMetadata(m *genericMetric) ([]byte, []byte, []byte, error) {
	resAttrs, err := json.Marshal(m.resMetadata.ResAttrs.AsRaw())
	if err != nil {
		return nil, nil, nil, err
	}

	scopeAttrs, err := json.Marshal(m.resMetadata.InstrScope.Attributes().AsRaw())
	if err != nil {
		return nil, nil, nil, err
	}

	metadata, err := json.Marshal(m.metadata.AsRaw())
	if err != nil {
		return nil, nil, nil, err
	}

	return resAttrs, scopeAttrs, metadata, nil
}

func exemplarsToRaw(exemplars pmetric.ExemplarSlice) []any {
	result := make([]any, 0, exemplars.Len())

	for i := 0; i < exemplars.Len(); i++ {
		exemplar := exemplars.At(i)

		raw := map[string]any{
			"timestamp":           exemplar.Timestamp().AsTime(),
			"filtered_attributes": exemplar.FilteredAttributes().AsRaw(),
			"trace_id":            exemplar.TraceID().String(),
			"span_id":             exemplar.SpanID().String(),
		}

		switch exemplar.ValueType() {
		case pmetric.ExemplarValueTypeInt:
			raw["value"] = exemplar.IntValue()
		case pmetric.ExemplarValueTypeDouble:
			raw["value"] = exemplar.DoubleValue()
		}

		result = append(result, raw)
	}

	return result
}

func quantileValuesToRaw(values pmetric.SummaryDataPointValueAtQuantileSlice) []any {
	result := make([]any, 0, values.Len())

	for i := 0; i < values.Len(); i++ {
		value := values.At(i)

		result = append(result, map[string]any{
			"quantile": value.Quantile(),
			"value":    value.Value(),
		})
	}

	return result
}
