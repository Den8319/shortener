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

const shortURLLength = 8

type DeleteRequest struct {
	Context   context.Context
	ShortURLs []string
	UserID    string
}

type Store struct {
	urls          map[string]string
	mu            sync.RWMutex
	loader        model.Loader
	DeleteQueue   chan DeleteRequest
	deleteWorkers int
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

func (s *Store) GetShortList(ctx context.Context, items []model.BatchRequestItem, userUUID string ) ([]model.BatchResponseItem, error) {
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

func (s *Store) GetUserURLs(ctx context.Context, userUUID string) ([]model.URL, error) {
	if s.loader == nil {
		return nil, fmt.Errorf("loader is not initialized")
	}
	return s.loader.GetUserURLs(ctx, userUUID)
}

func (s *Store) StartDeleter(ctx context.Context, workers int) {
	s.deleteWorkers = workers
	s.DeleteQueue = make(chan DeleteRequest, workers*2)

	for i := 0; i < workers; i++ {
		go s.deleteWorker(ctx)
	}
	log.Info().Int("workers", workers).Msg("started delete workers")
}

func (s *Store) deleteWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("delete worker shutting down")
			return
		case req := <-s.DeleteQueue:
			s.processDeleteRequest(req)
		}
	}
}

func (s *Store) processDeleteRequest(req DeleteRequest) {
	err := s.loader.Delete(context.Background(), req.ShortURLs, req.UserID)
	if err != nil {
		log.Error().Err(err).Str("user_id", req.UserID).Msg("failed to delete URLs")
	}
}

func (s *Store) DeleteURLs(ctx context.Context, shortURLs []string, userUUID string) error {
	if s.DeleteQueue == nil {
		return fmt.Errorf("delete queue is not initialized, call StartDeleter first")
	}

	req := DeleteRequest{
		Context:   ctx,
		ShortURLs: shortURLs,
		UserID:    userUUID,
	}

	select {
	case s.DeleteQueue <- req:
		return nil
	default:
		return fmt.Errorf("delete queue is full")
	}
}
