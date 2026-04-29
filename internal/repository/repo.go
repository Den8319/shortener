package repository

import (
	"context"

	"github.com/Den8319/shortener/internal/model"
)

// URLRepository — интерфейс для хранения URL
type URLRepository interface {
	Save(ctx context.Context, url *model.URL) error
	GetURL(ctx context.Context, short string) (*model.URL, error)
	GetAllURLs(ctx context.Context, userID string) ([]*model.URL, error)
	Close() error
}

// Pingable — интерфейс для проверки подключения
type Pingable interface {
	Ping(ctx context.Context) error
}
