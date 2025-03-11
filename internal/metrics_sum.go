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
	sumMetricTableInsertSQL = `
	INSERT INTO "%s"."%s" (
		resource_url, resource_attributes,
		scope_name, scope_version, scope_attributes, scope_dropped_attr_count, scope_url, service_name,
		name, description, unit,
		start_timestamp, timestamp,
		attribute1, attribute2, attribute3, attribute4, attribute5,
		attribute6, attribute7, attribute8, attribute9, attribute10,
		attribute11, attribute12, attribute13, attribute14, attribute15,
		attribute16, attribute17, attribute18, attribute19, attribute20,
		metadata,
		value, exemplars, flags, aggregation_temporality, is_monotonic
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39)
	`
)

var (
	sumMetricTableColumns = []string{
		"value DOUBLE PRECISION",

		"exemplars JSONB",
		"flags     INTEGER",

		"aggregation_temporality INTEGER",
		"is_monotonic            BOOLEAN",
	}
)

type sumMetric struct {
	resMetadata *ResourceMetadata

	sum         pmetric.Sum
	name        string
	description string
	unit        string
	metadata    pcommon.Map
}

type sumMetricsGroup struct {
	DBType     DBType
	SchemaName string

	metrics []*sumMetric
	count   int
}

func (g *sumMetricsGroup) Add(resMetadata *ResourceMetadata, metric any, name, description, unit string, metadata pcommon.Map) error {
	sum, ok := metric.(pmetric.Sum)
	if !ok {
		return fmt.Errorf("metric param is not Sum type")
	}

	g.count += sum.DataPoints().Len()
	g.metrics = append(g.metrics, &sumMetric{
		resMetadata: resMetadata,
		sum:         sum,
		name:        name,
		description: description,
		unit:        unit,
		metadata:    metadata,
	})

	return nil
}

func (g *sumMetricsGroup) insert(ctx context.Context, client *sql.DB) error {
	logger.Debug("Inserting sum metrics")

	if g.count == 0 {
		return nil
	}

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

			statement, err := tx.PrepareContext(ctx, fmt.Sprintf(sumMetricTableInsertSQL, g.SchemaName, m.name))
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

			for i := 0; i < m.sum.DataPoints().Len(); i++ {
				dp := m.sum.DataPoints().At(i)
				attrs := getAttributesAsSlice(dp.Attributes())

				tx.Stmt(statement).ExecContext(ctx,
					m.resMetadata.ResURL, resAttrs,
					m.resMetadata.InstrScope.Name(),
					m.resMetadata.InstrScope.Version(),
					scopeAttrs,
					m.resMetadata.InstrScope.DroppedAttributesCount(),
					m.resMetadata.ScopeUrl,
					getServiceName(m.resMetadata.ResAttrs),
					m.name, m.description, m.unit,
					dp.StartTimestamp().AsTime(), dp.Timestamp().AsTime(),
					attrs[0],  attrs[1],  attrs[2],  attrs[3],  attrs[4],
					attrs[5],  attrs[6],  attrs[7],  attrs[8],  attrs[9],
					attrs[10], attrs[11], attrs[12], attrs[13], attrs[14],
					attrs[15], attrs[16], attrs[17], attrs[18], attrs[19],
					metadata,
					getValue(dp.IntValue(), dp.DoubleValue(), dp.ValueType()),
					dp.Exemplars(),
					uint32(dp.Flags()),
					int32(m.sum.AggregationTemporality()),
					m.sum.IsMonotonic(),
				)
			}

			return nil
		})
		errs = errors.Join(errs, err)
	}
	if errs != nil {
		return fmt.Errorf("insert sum metrics failed: %w", errs)
	}

	return nil
}

func (g *sumMetricsGroup) createTable(ctx context.Context, client *sql.DB, metricName string) error {
	metricTableColumns := slices.Concat(getBaseMetricTableColumns(g.DBType), sumMetricTableColumns)

	return createMetricTable(ctx, client, g.SchemaName, metricName, metricTableColumns)
}
