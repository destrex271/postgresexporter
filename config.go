package postgresexporter

import (
	"database/sql"

	"github.com/destrex271/postgresexporter/internal/db"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

type Config struct {
	DatabaseConfig   DatabaseConfig               `mapstructure:"database"`

	LogsTableName    string                       `mapstructure:"logs_table_name"`
	TracesTableName  string                       `mapstructure:"traces_table_name"`
	MetricsTableName string                       `mapstructure:"metrics_table_name"`

	TimeoutSettings  exporterhelper.TimeoutConfig `mapstructure:",squash"`
	QueueSettings    exporterhelper.QueueConfig   `mapstructure:"sending_queue"`
}

type DatabaseConfig struct {
	Host     string                     `mapstructure:"host"`
	Port     int                        `mapstructure:"port"`
	Username string                     `mapstructure:"username"`
	Password string                     `mapstructure:"password"`
	Database string                     `mapstructure:"database"`
	SSLmode  string                     `mapstructure:"sslmode"`
}

func (cfg *Config) buildDB() (*sql.DB, error) {
	dbcfg := cfg.DatabaseConfig

	conn, err := db.Open(db.URL(dbcfg.Host, dbcfg.Port, dbcfg.Username, dbcfg.Password, dbcfg.Database, dbcfg.SSLmode))
	if err != nil {
		return nil, err
	}

	return conn, nil
}
