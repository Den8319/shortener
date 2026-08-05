package model

import (
	"context"
)

// Storage — интерфейс бизнес-логики (используется в handler).
type Storage interface {
	GetShortURL(ctx context.Context, longURL string, userUUID string) (string, error)
	GetLongURL(ctx context.Context, shortURL string) (string, error)
	GetShortList(ctx context.Context, items []BatchRequestItem, userUUID string) ([]BatchResponseItem, error)
	GetUserURLs(ctx context.Context, userUUID string) ([]URL, error)
	DeleteURLs(ctx context.Context, shortURLs []string, userUUID string) error
}

// Pinger — интерфейс для проверки доступности базы данных.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Loader — интерфейс доступа к данным (используется в service).
type Loader interface {
	Save(ctx context.Context, url *URL, userUUID string) error
	GetLongURL(ctx context.Context, shortURL string) (string, error)
	Load(ctx context.Context) (map[string]string, error)
	Close() error
	GetUserURLs(ctx context.Context, userUUID string) ([]URL, error)
	Delete(ctx context.Context, shortURLs []string, userUUID string) error
	Stats(ctx context.Context) (urls int, users int, err error)
}
