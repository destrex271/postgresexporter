package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func URL(host string, port int, user, password, dbname, sslmode string) string {
	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		user, url.QueryEscape(password), host, port, dbname, sslmode,
	)
}

func Open(databaseUrl string) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseUrl)
	if err != nil {
		return nil, fmt.Errorf("failed opening connection to postgres: %w", err)
	}

	return db, nil
}

func DoWithTx(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db.Begin: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}
