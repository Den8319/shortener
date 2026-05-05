package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Den8319/shortener/internal/model"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

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

	return &DB{
		conn: conn,
	}, nil
}

func (db *DB) Ping(ctx context.Context) error {
	return db.conn.PingContext(ctx)
}

func (db *DB) Create(ctx context.Context) error {

	_, err := db.conn.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS t_urls (n_id SERIAL PRIMARY KEY, s_long_url VARCHAR(1000) NOT NULL, s_short_url VARCHAR(50) NOT NULL);
		CREATE INDEX IF NOT EXISTS idx_data_full ON t_urls(s_long_url);
		CREATE INDEX IF NOT EXISTS idx_data_short ON t_urls(s_short_url);`)

	if err != nil {
		return err
	}
	return nil

}

func (db *DB) SaveBatch(ctx context.Context, urls []*model.URL) error {
	if len(urls) == 0 {
		return nil
	}

	if err := db.Ping(ctx); err != nil {
		//logger
		return err
	}

	if err := db.Create(ctx); err != nil {
		//logger
		return err
	}

	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := db.conn.PrepareContext(ctx, "INSERT INTO t_urls (s_short_url, s_long_url) VALUES ($1, $2)")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, url := range urls {
		_, err := stmt.ExecContext(ctx, url.ShortURL, url.LongURL)
		if err != nil {
			return fmt.Errorf("failed to execute statement: %w", err)

		
}
		}

		if err := tx.Commit(); err != nil {
         return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (db *DB) loadList(ctx context.Context, conn Connector) (map[string]string, error) {
	if err := db.Ping(ctx); err != nil {
		return nil, err
	}

	if err := db.Create(ctx); err != nil {
		//logger
		return nil, err
	}

	rows, err := conn.QueryContext(ctx, "SELECT s_long_url, s_short_url FROM t_urls")
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

func (db *DB) GetShort(ctx context.Context, conn Connector, longURL string) (string, error) {
	const query = `SELECT s_short_url FROM t_urls WHERE s_long_url = $1`
	var shortURL string

	row := conn.QueryRowContext(ctx, query, longURL)
	err := row.Scan(&shortURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("url not found")
		}
		return "", fmt.Errorf("scan row: %w", err)
	}
	return shortURL, nil
}

func (db *DB) Save(ctx context.Context, url *model.URL) error {
	_, err := db.conn.ExecContext(ctx, "INSERT INTO t_urls (s_short_url, s_long_url) VALUES ($1, $2)", url.ShortURL, url.LongURL)
	if err != nil {
		return fmt.Errorf("failed to save URL: %w", err)
	}
	return nil
}

func (db *DB) getLongURL(ctx context.Context, conn Connector, shortURL string) (string, error) {
	const query = `SELECT s_long_url FROM t_urls WHERE s_short_url = $1`
	var longURL string

	err := conn.QueryRowContext(ctx, query, shortURL).Scan(&longURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("url not found")
		}
		return "", fmt.Errorf("scan row: %w", err)
	}
	return longURL, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}
