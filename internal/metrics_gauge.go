package internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/destrex271/postgresexporter/internal/db"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

const (
	gaugeMetricTableInsertSQL = `
	INSERT INTO "%s"."%s" (
		resource_url, resource_attributes,
		scope_name, scope_version, scope_attributes, scope_dropped_attr_count, scope_url, service_name,
		name, type, description, unit,
		start_timestamp, timestamp,
		attribute1, attribute2, attribute3, attribute4, attribute5,
		attribute6, attribute7, attribute8, attribute9, attribute10,
		attribute11, attribute12, attribute13, attribute14, attribute15,
		attribute16, attribute17, attribute18, attribute19, attribute20,
		metadata,
		value, exemplars, flags
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38)
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
	MetricsType pmetric.MetricType

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

	if g.count == 0 {
		return nil
	}

	attributesMappings, err := GetAttributesMappingsByNames(ctx, client, g.SchemaName, g.getMetricsNames())
	if err != nil {
		return err
	}

	attributesMappingsMap := groupAttrsMappingsByName(attributesMappings)

	var errs error
	for _, m := range g.metrics {
		err := db.DoWithTx(ctx, client, func(tx *sql.Tx) error {
			exists, err := CheckIfTableExists(ctx, client, g.SchemaName, m.name)
			if err != nil {
				return err
			}

			if !exists {
				g.createTable(ctx, client, m.name)
			}

			statement, err := tx.PrepareContext(ctx, fmt.Sprintf(gaugeMetricTableInsertSQL, g.SchemaName, m.name))
			if err != nil {
				return err
			}

			defer func() {
				_ = statement.Close()
			}()

			resAttrs, err := json.Marshal(m.resMetadata.ResAttrs.AsRaw())
			if err != nil {
				return err
			}

			scopeAttrs, err := json.Marshal(m.resMetadata.InstrScope.Attributes().AsRaw())
			if err != nil {
				return err
			}

			metadata, err := json.Marshal(m.metadata.AsRaw())
			if err != nil {
				return err
			}

			for i := range m.gauge.DataPoints().Len() {
				dp := m.gauge.DataPoints().At(i)

				if dp.Timestamp().AsTime().IsZero() {
					errs = errors.Join(errs, fmt.Errorf("data points with the 0 value for TimeUnixNano SHOULD be rejected by consumers"))
					continue
				}

				attrsMapping, present := attributesMappingsMap[m.name]
				if !present {
					attrsMapping = AttributesMapping{Name: m.name}
					err = insertAttributesMapping(ctx, client, g.SchemaName, &attrsMapping)
					if err != nil {
						errs = errors.Join(errs, err)
						continue
					}

					attributesMappingsMap[attrsMapping.Name] = attrsMapping
				}

				attrs, updated, err := getAttributesAsSliceAndCheckIfUpdated(dp.Attributes(), &attrsMapping)
				if err != nil {
					errs = errors.Join(errs, err)
					continue
				}

				if updated {
					err = updateAttributesMapping(ctx, client, g.SchemaName, &attrsMapping)
					if err != nil {
						errs = errors.Join(errs, err)
						continue
					}
				}

				tx.Stmt(statement).ExecContext(ctx,
					m.resMetadata.ResURL, resAttrs,
					m.resMetadata.InstrScope.Name(),
					m.resMetadata.InstrScope.Version(),
					scopeAttrs,
					m.resMetadata.InstrScope.DroppedAttributesCount(),
					m.resMetadata.ScopeUrl,
					getServiceName(m.resMetadata.ResAttrs),
					m.name, int32(g.MetricsType), m.description, m.unit,
					dp.StartTimestamp().AsTime(), dp.Timestamp().AsTime(),
					attrs[0],  attrs[1],  attrs[2],  attrs[3],  attrs[4],
					attrs[5],  attrs[6],  attrs[7],  attrs[8],  attrs[9],
					attrs[10], attrs[11], attrs[12], attrs[13], attrs[14],
					attrs[15], attrs[16], attrs[17], attrs[18], attrs[19],
					metadata,
					getValue(dp.IntValue(), dp.DoubleValue(), dp.ValueType()),
					dp.Exemplars(),
					uint32(dp.Flags()),
				)
			}

			return nil
		})
		errs = errors.Join(errs, err)
	}
	if errs != nil {
		return fmt.Errorf("insert gauge metrics failed: %w", errs)
	}

	return nil
}

func (g *gaugeMetricsGroup) createTable(ctx context.Context, client *sql.DB, metricName string) error {
	metricTableColumns := slices.Concat(getBaseMetricTableColumns(g.DBType), gaugeMetricTableColumns)

	return createMetricTable(ctx, client, g.SchemaName, metricName, metricTableColumns, g.DBType)
}

func (g *gaugeMetricsGroup) getMetricsNames() []string {
	result := []string{}

	for _, m := range g.metrics {
		result = append(result, m.name)
	}

	return result
}
