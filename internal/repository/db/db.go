package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Den8319/shortener/internal/model"

	"github.com/jackc/pgx/v5"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"
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

func (db *DB) Migrate(ctx context.Context) error {
	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database before migration: %w", err)
	}

	goose.SetBaseFS(nil)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	if err := goose.Up(db.conn, "migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Info().Msg("Migrations applied successfully")
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
	query := `INSERT INTO t_urls (s_short_url, s_long_url) VALUES ($1, $2) ON CONFLICT (s_short_url) DO NOTHING RETURNING s_short_url`

	err := db.conn.QueryRowContext(ctx, query, url.ShortURL, url.LongURL).Scan(&url.ShortURL)
	if err == sql.ErrNoRows {
		log.Error().Err(err).Msg("ErrNoRows")
		existingShort, getErr := db.GetShort(ctx, db.conn, url.LongURL)
		if getErr != nil {
			return fmt.Errorf("failed to retrieve existing short URL: %w", getErr)
		}
		url.ShortURL = existingShort
		return model.ErrURLAlreadyExists // Специальная ошибка для обработки в хендлере
	}
	if err != nil {
		log.Error().Err(err).Msg("db ping failed")
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
