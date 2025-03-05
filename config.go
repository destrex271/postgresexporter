package postgresexporter

import (
	"database/sql"

	"github.com/destrex271/postgresexporter/internal/db"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

type Config struct {
	// Database config
	DatabaseConfig  DatabaseConfig               `mapstructure:"database"`

	// Decided to create separate tables for each metric to provide better performance.
	// But you can define DatabaseConfig.Schema in which to create it.
	// Metrics table name
	// MetricsTableName string                       `mapstructure:"metrics_table_name"`

	// Logs table name
	LogsTableName   string                       `mapstructure:"logs_table_name"`
	// Traces table name
	TracesTableName string                       `mapstructure:"traces_table_name"`

	// Pre-create the schema and tables if true. Default - true.
	CreateSchema    bool                         `mapstructure:"create_schema"`

	// Timeout
	TimeoutSettings exporterhelper.TimeoutConfig `mapstructure:",squash"`
	// Sending queue settings
	QueueSettings   exporterhelper.QueueConfig   `mapstructure:"sending_queue"`
}

type DatabaseConfig struct {
	// Type. Can be 'postgresql', 'timescaledb', 'paradedb' etc.
	// The organization of data may be different for each type
	Type     DBType `mapstructure:"type"`
	// Host
	Host     string             `mapstructure:"host"`
	// Port
	Port     int                `mapstructure:"port"`
	// Username
	Username string             `mapstructure:"username"`
	// Password
	Password string             `mapstructure:"password"`
	// Database name. Default - otel
	Database string             `mapstructure:"database"`
	// Schema name. Default - otel
	Schema   string             `mapstructure:"schema"`
	// SSL mode. Default - disabled
	SSLmode  string             `mapstructure:"sslmode"`
}

// Should create schema
func (cfg *Config) shouldCreateSchema() bool {
	return cfg.CreateSchema
}

// Build database connection
func (cfg *Config) buildDB() (*sql.DB, error) {
	dbcfg := cfg.DatabaseConfig

	conn, err := db.Open(db.URL(dbcfg.Host, dbcfg.Port, dbcfg.Username, dbcfg.Password, dbcfg.Database, dbcfg.SSLmode))
	if err != nil {
		return nil, err
	}

	return conn, nil
}
