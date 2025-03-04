package postgresexporter

import (
	"database/sql"

	"github.com/destrex271/postgresexporter/internal/db"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

type Config struct {
	// Database config
	DatabaseConfig   DatabaseConfig               `mapstructure:"database"`

	// Metrics table name
	MetricsTableName string                       `mapstructure:"metrics_table_name"`
	// Logs table name
	LogsTableName    string                       `mapstructure:"logs_table_name"`
	// Traces table name
	TracesTableName  string                       `mapstructure:"traces_table_name"`

	// Timeout
	TimeoutSettings  exporterhelper.TimeoutConfig `mapstructure:",squash"`
	// Sending queue settings
	QueueSettings    exporterhelper.QueueConfig   `mapstructure:"sending_queue"`
}

type DatabaseConfig struct {
	// Host
	Host     string                     `mapstructure:"host"`
	// Port
	Port     int                        `mapstructure:"port"`
	// Username
	Username string                     `mapstructure:"username"`
	// Password
	Password string                     `mapstructure:"password"`
	// Database name
	Database string                     `mapstructure:"database"`
	// SSL mode
	SSLmode  string                     `mapstructure:"sslmode"`
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
