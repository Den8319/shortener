// Package db содержит реализацию хранилища на PostgreSQL.
package db

import (
	"context"
	"database/sql"
	"fmt"
)

// DB — структура для работы с базой данных PostgreSQL.
// Содержит подключение к базе данных и предоставляет методы для выполнения операций с URL.
type DB struct {
	conn *sql.DB
}

// Connector — интерфейс для абстракции работы с SQL-запросами.
// Позволяет использовать как *sql.DB, так и *sql.Tx в методах репозитория.
type Connector interface {
	// QueryContext выполняет SQL-запрос и возвращает строки результатов.
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	// ExecContext выполняет SQL-запрос без возврата строк.
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	// QueryRowContext выполняет SQL-запрос и возвращает одну строку результата.
	QueryRowContext(context.Context, string, ...any) *sql.Row
	// PrepareContext подготавливает SQL-запрос для последующего выполнения.
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

// Load загружает все URL из базы данных в оперативную память.
// Выполняет проверку подключения, применение миграций и чтение данных.
// Возвращает карту соответствия коротких URL длинным (short -> long).
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

// GetShortURL находит короткий URL по его полному (длинному) варианту.
// Выполняется внутри транзакции для обеспечения согласованности данных.
// Возвращает короткий URL в виде строки или ошибку, если URL не найден.
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

// GetLongURL находит полный (длинный) URL по его короткому варианту.
// Проверяет, не помечен ли URL как удалённый (soft delete).
// Выполняется внутри транзакции для обеспечения согласованности данных.
// Возвращает длинный URL в виде строки или ошибку, если URL не найден или удалён.
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
