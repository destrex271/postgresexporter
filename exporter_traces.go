package postgresexporter

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/destrex271/postgresexporter/internal/db"
	"github.com/destrex271/postgresexporter/internal/traceutil"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/ptrace"
	conventions "go.opentelemetry.io/collector/semconv/v1.27.0"
	"go.uber.org/zap"
)

type tracesExporter struct {
	client    *sql.DB
	insertSQL string
	logger    *zap.Logger
	cfg       *Config
}

func newTracesExporter(logger *zap.Logger, cfg *Config) (*tracesExporter, error) {
	dbcfg := cfg.DatabaseConfig

	client, err := db.Open(db.URL(dbcfg.Host, dbcfg.Port, dbcfg.Username, dbcfg.Password, dbcfg.Database, dbcfg.SSLmode))
	if err != nil {
		return nil, err
	}

	return &tracesExporter{
		client:    client,
		insertSQL: renderInsertTracesSQL(cfg),
		logger:    logger,
		cfg:       cfg,
	}, nil
}

func (e *tracesExporter) start(ctx context.Context, _ component.Host) error {
	return createTracesTable(ctx, e.cfg, e.client)
}

func (e *tracesExporter) shutdown(_ context.Context) error {
	if e.client != nil {
		return e.client.Close()
	}
	return nil
}

func convertEvents(events ptrace.SpanEventSlice) string {
	var eventString []string
	for i := 0; i < events.Len(); i++ {
		event := events.At(i)
		time := event.Timestamp().String()
		name := event.Name()
		attr := attributesToMap(event.Attributes())
		evt := fmt.Sprintf("{%s, %s, %s}", time, name, attr)
		eventString = append(eventString, evt)
	}
	return marshalSliceToString(eventString)
}

func convertLinks(links ptrace.SpanLinkSlice) string {
	var linksData []string
	for i := 0; i < links.Len(); i++ {
		link := links.At(i)
		traceID := traceutil.TraceIDToHexOrEmptyString(link.TraceID())
		spanIDs := traceutil.SpanIDToHexOrEmptyString(link.SpanID())
		states := link.TraceState().AsRaw()
		attrs := attributesToMap(link.Attributes())
		lnk := fmt.Sprintf("{%s, %s, %s, %s}", traceID, spanIDs, states, attrs)
		linksData = append(linksData, lnk)
	}
	return marshalSliceToString(linksData)
}

func (e *tracesExporter) pushTraceData(ctx context.Context, td ptrace.Traces) error {
	start := time.Now()
	err := doWithTx(ctx, e.client, func(tx *sql.Tx) error {
		statement, err := tx.PrepareContext(ctx, e.insertSQL)
		if err != nil {
			return fmt.Errorf("PrepareContext:%w", err)
		}
		defer func() {
			_ = statement.Close()
		}()
		for i := 0; i < td.ResourceSpans().Len(); i++ {
			spans := td.ResourceSpans().At(i)
			res := spans.Resource()
			resAttr := attributesToMap(res.Attributes())
			var serviceName string
			if v, ok := res.Attributes().Get(conventions.AttributeServiceName); ok {
				serviceName = v.Str()
			}
			for j := 0; j < spans.ScopeSpans().Len(); j++ {
				rs := spans.ScopeSpans().At(j).Spans()
				scopeName := spans.ScopeSpans().At(j).Scope().Name()
				scopeVersion := spans.ScopeSpans().At(j).Scope().Version()
				for k := 0; k < rs.Len(); k++ {
					r := rs.At(k)
					spanAttr := attributesToMap(r.Attributes())
					status := r.Status()
					events := convertEvents(r.Events())
					links := convertLinks(r.Links())
					_, err = statement.ExecContext(ctx,
						r.StartTimestamp().AsTime(),
						traceutil.TraceIDToHexOrEmptyString(r.TraceID()),
						traceutil.SpanIDToHexOrEmptyString(r.SpanID()),
						traceutil.SpanIDToHexOrEmptyString(r.ParentSpanID()),
						r.TraceState().AsRaw(),
						r.Name(),
						r.Kind().String(),
						serviceName,
						resAttr,
						scopeName,
						scopeVersion,
						spanAttr,
						r.EndTimestamp().AsTime().Sub(r.StartTimestamp().AsTime()).Nanoseconds(),
						status.Code().String(),
						status.Message(),
						events,
						links,
					)
					if err != nil {
						return fmt.Errorf("ExecContext:%w", err)
					}
				}
			}
		}
		return nil
	})
	duration := time.Since(start)
	e.logger.Debug("insert traces", zap.Int("records", td.SpanCount()),
		zap.String("cost", duration.String()))
	return err
}

// SQL Content from below
const (
	// language=PostgreSQL
	createTracesTableSQL = `
	CREATE TABLE IF NOT EXISTS %s (
		"Timestamp" TIMESTAMP(9) NOT NULL,
		"TraceId" TEXT,
		"SpanId" TEXT,
		"ParentSpanId" TEXT,
		"TraceState" TEXT,
		"SpanName" TEXT,
		"SpanKind" TEXT,
		"ServiceName" TEXT,
		"ResourceAttributes" JSONB,
		"ScopeName" TEXT,
		"ScopeVersion" TEXT,
		"SpanAttributes" JSONB,
		"Duration" BIGINT,
		"StatusCode" TEXT,
		"StatusMessage" TEXT,
		"Events" JSONB, -- Using JSONB to store the Nested events structure
		"Links" JSONB,  -- Using JSONB to store the Nested links structure

		PRIMARY KEY ("ServiceName", "SpanName", "Timestamp")
	);
`
	// language=PostgreSQL
	insertTracesSQLTemplate = `
	INSERT INTO %s (
		"Timestamp",
		"TraceId",
		"SpanId",
		"ParentSpanId",
		"TraceState",
		"SpanName",
		"SpanKind",
		"ServiceName",
		"ResourceAttributes",
		"ScopeName",
		"ScopeVersion",
		"SpanAttributes",
		"Duration",
		"StatusCode",
		"StatusMessage",
		"Events",
		"Links"
	) VALUES (
		$1, -- "Timestamp"
		$2, -- "TraceId"
		$3, -- "SpanId"
		$4, -- "ParentSpanId"
		$5, -- "TraceState"
		$6, -- "SpanName"
		$7, -- "SpanKind"
		$8, -- "ServiceName"
		$9, -- "ResourceAttributes" (JSONB)
		$10, -- "ScopeName"
		$11, -- "ScopeVersion"
		$12, -- "SpanAttributes" (JSONB)
		$13, -- "Duration"
		$14, -- "StatusCode"
		$15, -- "StatusMessage"
		$16, -- "Events" (JSONB)
		$17  -- "Links" (JSONB)
	);

	`
)

const (
	createTraceIDTsTableSQL = `
	CREATE TABLE IF NOT EXISTS %s_trace_id_ts (
		"TraceId" TEXT,
		"Start" TIMESTAMP,
		"End" TIMESTAMP,

		PRIMARY KEY ("TraceId", "Start")
	);
`
	createTraceIDTsMaterializedViewSQL = `
	CREATE MATERIALIZED VIEW IF NOT EXISTS %s_trace_id_ts_mv AS
	SELECT
		"TraceId",
		MIN("Timestamp") AS "Start",
		MAX("Timestamp") AS "End"
	FROM %s
	WHERE "TraceId" != ''
	GROUP BY "TraceId";
	`
)

func createTracesTable(ctx context.Context, cfg *Config, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, renderCreateTracesTableSQL(cfg)); err != nil {
		return fmt.Errorf("exec create traces table sql: %w", err)
	}
	if _, err := db.ExecContext(ctx, renderCreateTraceIDTsTableSQL(cfg)); err != nil {
		return fmt.Errorf("exec create traceID timestamp table sql: %w", err)
	}
	mv_str := renderTraceIDTsMaterializedViewSQL(cfg)
	log.Println(mv_str)
	if _, err := db.ExecContext(ctx, renderTraceIDTsMaterializedViewSQL(cfg)); err != nil {
		return fmt.Errorf("exec create traceID timestamp view sql: %w", err)
	}
	return nil
}

func renderInsertTracesSQL(cfg *Config) string {
	return fmt.Sprintf(insertTracesSQLTemplate, cfg.TracesTableName)
}

func renderCreateTracesTableSQL(cfg *Config) string {
	return fmt.Sprintf(createTracesTableSQL, cfg.TracesTableName)
}

func renderCreateTraceIDTsTableSQL(cfg *Config) string {
	return fmt.Sprintf(createTraceIDTsTableSQL, cfg.TracesTableName)
}

func renderTraceIDTsMaterializedViewSQL(cfg *Config) string {
	return fmt.Sprintf(createTraceIDTsMaterializedViewSQL, cfg.TracesTableName, cfg.TracesTableName)
}
