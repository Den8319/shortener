package model

import (
	"context"
)

// Storage — интерфейс бизнес-логики (используется в handler)
type Storage interface {
	GetShortURL(ctx context.Context, longURL string, userUUID string) (string, error)
	GetLongURL(ctx context.Context, shortURL string) (string, error)
	GetShortList(ctx context.Context, items []BatchRequestItem) ([]BatchResponseItem, error)
	GetUserURLs(ctx context.Context, userUUID string) ([]URL, error)
}

type Pinger interface {
	Ping(ctx context.Context) error
}

// Loader — интерфейс доступа к данным (используется в service)
type Loader interface {
	Save(ctx context.Context, url *URL, userUUID string) error
	GetLongURL(ctx context.Context, shortURL string) (string, error)
	Load(ctx context.Context) (map[string]string, error)
	Close() error
	GetUserURLs(ctx context.Context, userUUID string) ([]URL, error)
}

// BatchSaver — опциональный интерфейс для пакетного сохранения
type BatchSaver interface {
	SaveBatch(ctx context.Context, urls []URL) error
}
