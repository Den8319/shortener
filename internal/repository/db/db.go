package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Den8319/shortener/internal/model"

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
	queries := []string{
		"CREATE TABLE IF NOT EXISTS t_urls (n_id SERIAL PRIMARY KEY, s_long_url VARCHAR(1000) NOT NULL, s_short_url VARCHAR(50) NOT NULL);",
		"CREATE INDEX IF NOT EXISTS idx_data_full ON t_urls(s_long_url);",
		"CREATE INDEX IF NOT EXISTS idx_data_short ON t_urls(s_short_url);",
	}
	for _, query := range queries {
		_, err := db.conn.ExecContext(ctx, query)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) Save(ctx context.Context, url *model.URL) error {
	const query = `INSERT INTO t_urls (s_long_url, s_short_url) VALUES ($1, $2)`
	_, err := db.conn.ExecContext(ctx, query, url.LongURL, url.ShortURL)
	if err != nil {
		return fmt.Errorf("insert url pair: %w", err)
	}
	return nil
}

func (db *DB) GetURL(ctx context.Context, short string) (*model.URL, error) {
	const query = `SELECT s_short_url, s_long_url FROM t_urls WHERE s_short_url = $1`

	var url model.URL
	if err := db.conn.QueryRowContext(ctx, query, short).Scan(&url.ShortURL, &url.LongURL); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("url not found")
		}
		return nil, fmt.Errorf("scan row: %w", err)
	}
	return &url, nil
}

func (db *DB) GetAllURLs(ctx context.Context, userID string) ([]*model.URL, error) {
	const query = `SELECT s_short_url, s_long_url FROM t_urls`

	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("select all urls: %w", err)
	}
	defer rows.Close()

	var urls []*model.URL
	for rows.Next() {
		var url model.URL
		if err := rows.Scan(&url.ShortURL, &url.LongURL); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		urls = append(urls, &url)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return urls, nil
}

func (db *DB) Load(ctx context.Context) (map[string]string, error) {
	if err := db.Ping(ctx); err != nil {
		return nil, err
	}

	rows, err := db.conn.QueryContext(ctx, "SELECT s_long_url, s_short_url FROM t_urls")
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

func (db *DB) Close() error {
	return db.conn.Close()
}