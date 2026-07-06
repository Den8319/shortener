package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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

func (db *DB) Save(ctx context.Context, url *model.URL, userUUID string) error {
	query := `INSERT INTO t_urls (s_short_url, s_long_url, u_user) VALUES ($1, $2, $3) ON CONFLICT (s_short_url) DO NOTHING RETURNING s_short_url`

	log.Info().
		Str("short_url", url.ShortURL).
		Str("long_url", url.LongURL).
		Str("user_id", userUUID).
		Msg("saving URL to database")

	err := db.conn.QueryRowContext(ctx, query, url.ShortURL, url.LongURL, userUUID).Scan(&url.ShortURL)
	if err == sql.ErrNoRows {
		log.Error().Err(err).Msg("ErrNoRows")
		existingShort, getErr := db.GetShort(ctx, db.conn, url.LongURL)
		if getErr != nil {
			return fmt.Errorf("failed to retrieve existing short URL: %w", getErr)
		}
		url.ShortURL = existingShort
		return model.ErrURLAlreadyExists // Специальная ошибка для обработки в хендлерах
	}
	if err != nil {
		log.Error().Err(err).Msg("db ping failed")
		return fmt.Errorf("failed to save URL: %w", err)
	}
	return nil
}

func (db *DB) GetLong(ctx context.Context, conn Connector, shortURL string) (string, error) {
	const query = `SELECT s_long_url, b_deleted FROM t_urls WHERE s_short_url = $1`
	var longURL string
	var isDeleted bool

	err := conn.QueryRowContext(ctx, query, shortURL).Scan(&longURL, &isDeleted)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("url not found")
		}
		return "", fmt.Errorf("scan row: %w", err)
	}
	if isDeleted {
		return "", model.ErrURLDeleted
	}
	return longURL, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) GetUserURLs(ctx context.Context, userUUID string) ([]model.URL, error) {
	if err := db.Ping(ctx); err != nil {
		return nil, err
	}

	query := `SELECT s_short_url, s_long_url FROM t_urls WHERE u_user = $1`
	rows, err := db.conn.QueryContext(ctx, query, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user URLs: %w", err)
	}
	defer rows.Close()

	var urls []model.URL
	for rows.Next() {
		var shortURL, longURL string
		if err := rows.Scan(&shortURL, &longURL); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		urls = append(urls, model.URL{ShortURL: shortURL, LongURL: longURL})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return urls, nil
}

func (db *DB) Delete(ctx context.Context, shortURLs []string, userUUID string) error {

	log.Info().Int("count", len(shortURLs)).Str("user_id", userUUID).Msg("starting soft delete batch")

	if len(shortURLs) == 0 {
		return nil
	}

	// Формируем список идентификаторов для IN-запроса
	placeholders := make([]string, len(shortURLs))
	args := make([]any, len(shortURLs)+1)
	for i, s := range shortURLs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = s
	}
	args[len(shortURLs)] = userUUID

	query := fmt.Sprintf(`UPDATE t_urls SET b_deleted = true WHERE s_short_url IN (%s) AND u_user = $%d`,
		strings.Join(placeholders, ","), len(shortURLs)+1)

	result, err := db.conn.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute delete query: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	log.Info().Int("deleted", int(rowsAffected)).Msg("Soft delete batch completed")
	return nil
}
