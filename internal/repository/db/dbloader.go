package db

import (
	"context"
	"database/sql"
	"fmt"
)

type DB struct {
	conn *sql.DB
}

type Connector interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

func (db *DB) Load(ctx context.Context) (map[string]string, error) {
	if err := db.Ping(ctx); err != nil {
		//logger
		return nil, err
	}

	if err := db.Migrate(ctx); err != nil {
		//logger
		return nil, err
	}

	return db.loadList(ctx, db.conn)
}

func (db *DB) GetShortURL(ctx context.Context, longURL string) (string, error) {

	if err := db.Ping(ctx); err != nil {
		//logger
		return "", err
	}

	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		//logger
		return "", err
	}
	defer tx.Rollback()

	short, err := db.GetShort(ctx, tx, longURL)
	if err != nil {
		//logger
		tx.Rollback()
		return "", err
	}

	return short, tx.Commit()
}

func (db *DB) GetLongURL(ctx context.Context, shortURL string) (string, error) {
	if err := db.Ping(ctx); err != nil {
		return "", err
	}

	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	longURL, err := db.GetLong(ctx, tx, shortURL)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	return longURL, nil
}
