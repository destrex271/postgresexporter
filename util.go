package postgresexporter

import (
	"context"
	"database/sql"
	"fmt"
)

func createSchema(ctx context.Context, client *sql.DB, config *Config) error {
	query := "CREATE SCHEMA IF NOT EXISTS ?"
	_, err := client.ExecContext(ctx, query, config.DatabaseConfig.Schema)
	if err != nil {
		return fmt.Errorf("failed creating schema: %w", err)
	}

	return nil
}
