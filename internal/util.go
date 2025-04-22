package internal

import (
	"context"
	"database/sql"
	"fmt"

	"go.uber.org/zap"
)

type DBType string

const (
	DBTypePostgreSQL  DBType = "postgresql"
	DBTypeTimescaleDB DBType = "timescaledb"
	DBTypeParadeDB    DBType = "paradedb"

	createTableIfNotExistsSQL = `CREATE TABLE IF NOT EXISTS "%s"."%s" (%s)`
)

var logger *zap.Logger

func SetLogger(l *zap.Logger) {
	logger = l
}

func CheckIfTableExists(ctx context.Context, client *sql.DB, schemaName string, tableName string) (bool, error) {
	query := `SELECT EXISTS (SELECT 1 FROM pg_tables WHERE schemaname = $1 AND tablename = $2)`

	var exists bool
	err := client.QueryRowContext(ctx, query, schemaName, tableName).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func CreateSchema(ctx context.Context, client *sql.DB, schemaName string) error {
	query := `CREATE SCHEMA IF NOT EXISTS %s`
	_, err := client.ExecContext(ctx, fmt.Sprintf(query, schemaName))
	if err != nil {
		return fmt.Errorf("failed creating schema: %w", err)
	}

	return nil
}
