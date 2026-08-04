// Package store реализует сервис хранения сокращённых URL с кэшированием в памяти
// и поддержкой персистентного хранилища через интерфейс model.Loader.
package store

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/Den8319/shortener/internal/model"
	"github.com/Den8319/shortener/pkg/generator"
	"github.com/rs/zerolog/log"
)

// shortURLLength длина генерируемого короткого URL
const shortURLLength = 8

// DeleteRequest запрос на асинхронное удаление URL.
type DeleteRequest struct {
	Context   context.Context
	ShortURLs []string
	UserID    string
}

// Store сервис хранения URL с кэшем в памяти.
type Store struct {
	urls          map[string]string
	mu            sync.RWMutex
	loader        model.Loader
	DeleteQueue   chan DeleteRequest
	deleteWorkers int
	deleteWg      sync.WaitGroup
}

// New создаёт новый Store и загружает данные из loader.
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

// GetShortURL возвращает или создаёт короткий URL для longURL.
// Если URL уже существует, возвращает ErrURLAlreadyExists.
func (s *Store) GetShortURL(ctx context.Context, longURL string, userUUID string) (string, error) {
	log.Info().Msg("GetShortURL")
	s.mu.Lock()
	defer s.mu.Unlock()

	short, isNew, err := s.findOrGenerate(longURL)
	if err != nil {
		return "", err
	}
	if !isNew {
		return short, model.ErrURLAlreadyExists
	}
	log.Info().Bool("isNew:", isNew).Msg("GetShortURL")

	if s.loader != nil {
		url := &model.URL{ShortURL: short, LongURL: longURL}
		log.Info().
			Str("long_url", longURL).
			Str("userUUID", userUUID).
			Msg("Запись в БД")
		if err := s.loader.Save(ctx, url, userUUID); err != nil {
			if errors.Is(err, model.ErrURLAlreadyExists) {
				return url.ShortURL, model.ErrURLAlreadyExists
			}
			return "", fmt.Errorf("failed to save URL: %w", err)
		}
	}
	log.Info().Msgf("Saving to memory: %s -> %s", short, longURL)
	s.urls[short] = longURL
	return short, nil
}

// GetLongURL возвращает длинный URL по короткому идентификатору.
func (s *Store) GetLongURL(ctx context.Context, shortURL string) (string, error) {
	s.mu.RLock()
	long, ok := s.urls[shortURL]
	if ok {
		s.mu.RUnlock()
		return long, nil
	}
	s.mu.RUnlock()

	if s.loader != nil {
		url, err := s.loader.GetLongURL(ctx, shortURL)
		if err != nil {
			if errors.Is(err, model.ErrURLDeleted) {
				return "", err
			}
			return "", err
		}

		s.mu.Lock()
		s.urls[shortURL] = url
		s.mu.Unlock()

		return url, nil
	}

	return "", fmt.Errorf("short URL not found")
}

// Close закрывает хранилище и освобождает ресурсы.
func (s *Store) Close() error {
	if s.loader != nil {
		return s.loader.Close()
	}
	return nil
}

// findOrGenerate ищет существующий короткий URL или генерирует новый.
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

// GetShortList обрабатывает пакетный запрос на сокращение URL.
func (s *Store) GetShortList(ctx context.Context, items []model.BatchRequestItem, userUUID string) ([]model.BatchResponseItem, error) {
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

	for _, url := range toSave {
		_ = s.loader.Save(ctx, &url, userUUID)
	}

	for _, url := range toSave {
		s.urls[url.ShortURL] = url.LongURL
	}

	return results, nil
}

// GetUserURLs возвращает все URL пользователя.
func (s *Store) GetUserURLs(ctx context.Context, userUUID string) ([]model.URL, error) {
	if s.loader == nil {
		return nil, fmt.Errorf("loader is not initialized")
	}
	return s.loader.GetUserURLs(ctx, userUUID)
}

// StartDeleter запускает пул воркеров для асинхронного удаления URL.
// Воркеры завершаются при закрытии канала DeleteQueue (через CloseDeleter).
func (s *Store) StartDeleter(workers int) {
	s.deleteWorkers = workers
	s.DeleteQueue = make(chan DeleteRequest, workers*2)
	s.deleteWg.Add(workers)

	for range workers {
		go s.deleteWorker()
	}
	log.Info().Int("workers", workers).Msg("started delete workers")
}

// CloseDeleter закрывает канал очереди удаления и ожидает завершения воркеров.
func (s *Store) CloseDeleter() {
	if s.DeleteQueue == nil {
		return
	}
	close(s.DeleteQueue)
	s.deleteWg.Wait()
}

// deleteWorker воркер, обрабатывающий запросы на удаление.
func (s *Store) deleteWorker() {
	defer s.deleteWg.Done()
	for req := range s.DeleteQueue {
		s.processDeleteRequest(req)
	}
	log.Info().Msg("delete worker shutting down")
}

// processDeleteRequest обрабатывает один запрос на удаление URL.
func (s *Store) processDeleteRequest(req DeleteRequest) {
	err := s.loader.Delete(context.Background(), req.ShortURLs, req.UserID)
	if err != nil {
		log.Error().Err(err).Str("user_id", req.UserID).Msg("не удалось удалить URL")
		return
	}

	s.mu.Lock()
	for _, shortURL := range req.ShortURLs {
		delete(s.urls, shortURL)
	}
	s.mu.Unlock()
}

// DeleteURLs добавляет запрос на асинхронное удаление в очередь.
// Перед вызовом необходимо запустить воркеры через StartDeleter.
func (s *Store) DeleteURLs(ctx context.Context, shortURLs []string, userUUID string) error {
	if s.DeleteQueue == nil {
		return fmt.Errorf("delete queue is not initialized, call StartDeleter first")
	}

	req := DeleteRequest{
		Context:   ctx,
		ShortURLs: shortURLs,
		UserID:    userUUID,
	}

	s.DeleteQueue <- req
	return nil
}
