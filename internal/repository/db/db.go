package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DB struct {
	conn *sql.DB
}

func New(ctx context.Context, dsn string) (*DB, error) {
	if _, err := pgx.ParseConfig(dsn); err != nil {
		return nil, fmt.Errorf("invalid dsn: %w", err)
	}

	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db connection: %w", err)
	}

	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, err
	}

	return &DB{conn: conn}, nil
}

func (db *DB) Ping(ctx context.Context) error {
	return db.conn.PingContext(ctx)
}

func (db *DB) Migrate(ctx context.Context) error {
    _, err := db.conn.ExecContext(ctx, 
		`CREATE TABLE IF NOT EXISTS t_urls (
		n_id SERIAL PRIMARY KEY,
		s_long_url VARCHAR(1000) NOT NULL,
		s_short_url VARCHAR(50) NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_data_full ON t_urls(s_long_url);
	CREATE INDEX IF NOT EXISTS idx_data_short ON t_urls(s_short_url); 
    `)
    return err
}

func (db *DB) Load(ctx context.Context) (map[string]string, error) {

	if err := db.Ping(ctx); err != nil {
		return nil, err
	}

	rows, err := db.conn.QueryContext(ctx, "select s_long_url, s_short_url from t_urls")
	if err != nil {
		return nil, err
	}
	defer rows.Close()


	urls := make(map[string]string)
	for rows.Next() {
		var longURL, shortURL string
		if err := rows.Scan(&longURL, &shortURL); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		urls[shortURL] = longURL
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return urls, nil

}

func (db *DB) Save(ctx context.Context, shortURL, longURL string) error {
	const query = `INSERT INTO t_urls (s_long_url, s_short_url) VALUES ($1, $2)`
	_, err := db.conn.ExecContext(ctx, query, longURL, shortURL)
	if err != nil {
		return fmt.Errorf("insert url pair: %w", err)
	}
	return nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}
