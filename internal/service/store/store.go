package store

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/Den8319/shortener/internal/model"
	"github.com/Den8319/shortener/pkg/generator"
)

const shortURLLength = 8

type Store struct {
	urls   map[string]string
	mu     sync.RWMutex
	loader model.Loader
}

func New(loader model.Loader) (*Store, error) {
	s := &Store{
		urls:   make(map[string]string),
		loader: loader,
	}

	if loader != nil {
		urls, err := loader.Load(context.Background())
		if err != nil {
			return nil, err
		}
		s.urls = urls
	}

	return s, nil
}

func (s *Store) GetShortURL(ctx context.Context, longURL string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	short, isNew, err := s.findOrGenerate(longURL)
	if err != nil {
		return "", err
	}
	if !isNew {
		return short, nil
	}

	s.urls[short] = longURL

	if s.loader != nil {
		url := &model.URL{ShortURL: short, LongURL: longURL}
		if err := s.loader.Save(ctx, url); err != nil {
		
			if errors.Is(err, model.ErrURLAlreadyExists) {
				return url.ShortURL, model.ErrURLAlreadyExists
			}
			return "", fmt.Errorf("failed to save URL: %w", err)
		}
	}

	return short, nil
}

func (s *Store) GetLongURL(ctx context.Context, shortURL string) (string, error) {
	s.mu.RLock()
	// Сначала проверяем кэш в памяти
	long, ok := s.urls[shortURL]
	if ok {
		s.mu.RUnlock()
		return long, nil
	}
	s.mu.RUnlock()

	// Если не найдено в памяти, ищем в хранилище
	if s.loader != nil {
		url, err := s.loader.GetLongURL(ctx, shortURL)
		if err != nil {
			return "", err
		}

		// Обновляем кэш
		s.mu.Lock()
		s.urls[shortURL] = url
		s.mu.Unlock()

		return url, nil
	}

	return "", fmt.Errorf("short URL not found")
}

func (s *Store) Close() error {
	if s.loader != nil {
		return s.loader.Close()
	}
	return nil
}

func (s *Store) findOrGenerate(longURL string) (short string, isNew bool, err error) {
	for sh, stored := range s.urls {
		if stored == longURL {
			return sh, false, nil
		}
	}
	short, err = generator.GenerateShort(shortURLLength)
	if err != nil {
		return "", false, err
	}
	return short, true, nil
}

func (s *Store) GetShortList(ctx context.Context, items []model.BatchRequestItem) ([]model.BatchResponseItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var results []model.BatchResponseItem
	var toSave []model.URL

	for _, item := range items {
		short, isNew, err := s.findOrGenerate(item.LongURL)
		if err != nil {
			continue
		}

		results = append(results, model.BatchResponseItem{
			Corr:     item.Corr,
			ShortURL: short,
		})

		if isNew {
			toSave = append(toSave, model.URL{
				ShortURL: short,
				LongURL:  item.LongURL,
			})
		}
	}

	// Проверяем, поддерживает ли loader SaveBatch
	if len(toSave) > 0 && s.loader != nil {
		if batchSaver, ok := s.loader.(interface {
			SaveBatch(context.Context, []model.URL) error
		}); ok {
			_ = batchSaver.SaveBatch(ctx, toSave)
		} else {
			// Сохраняем по одному
			for _, url := range toSave {
				_ = s.loader.Save(ctx, &url)
			}
		}

		// Обновляем кэш
		for _, url := range toSave {
			s.urls[url.ShortURL] = url.LongURL
		}
	}

	return results, nil
}
