// Package service реализует сервисный слой
package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Den8319/shortener/internal/auth"
	"github.com/Den8319/shortener/internal/model"
	"github.com/rs/zerolog/log"
)

// ShortenResult содержит результат операции сокращения URL.
type ShortenResult struct {
	ShortURL string
	Created  bool
}

// Service инкапсулирует бизнес-логику сокращения URL.
type Service struct {
	storage model.Storage
}

// New создаёт сервис с указанным хранилищем.
func New(storage model.Storage) *Service {
	return &Service{storage: storage}
}

// Authenticate извлекает идентификатор пользователя из JWT-токена.
func (s *Service) Authenticate(token string) (string, error) {
	userUUID := auth.GetUser(token)
	if userUUID == "" {
		return "", model.ErrUnauthenticated
	}
	return userUUID, nil
}

// Shorten валидирует URL и создаёт или возвращает существующий короткий URL.
func (s *Service) Shorten(ctx context.Context, longURL, userUUID string) (*ShortenResult, error) {
	if userUUID == "" {
		return nil, model.ErrUnauthenticated
	}
	if !isValidURL(longURL) {
		return nil, model.ErrInvalidURL
	}

	shortURL, err := s.storage.GetShortURL(ctx, longURL, userUUID)
	if err != nil {
		if errors.Is(err, model.ErrURLAlreadyExists) {
			return &ShortenResult{ShortURL: shortURL, Created: false}, nil
		}
		log.Error().Err(err).Str("url", longURL).Msg("failed to shorten url")
		return nil, fmt.Errorf("failed to shorten url: %w", err)
	}

	return &ShortenResult{ShortURL: shortURL, Created: true}, nil
}

// Expand возвращает длинный URL по короткому идентификатору.
func (s *Service) Expand(ctx context.Context, shortID string) (string, error) {
	if shortID == "" {
		return "", model.ErrEmptyID
	}

	longURL, err := s.storage.GetLongURL(ctx, shortID)
	if err != nil {
		if errors.Is(err, model.ErrURLDeleted) {
			return "", model.ErrURLDeleted
		}
		log.Error().Err(err).Str("id", shortID).Msg("failed to get long url")
		return "", model.ErrURLNotFound
	}

	return longURL, nil
}

// ListUserURLs возвращает все URL пользователя.
func (s *Service) ListUserURLs(ctx context.Context, userUUID string) ([]model.URL, error) {
	if userUUID == "" {
		return nil, model.ErrUnauthenticated
	}

	urls, err := s.storage.GetUserURLs(ctx, userUUID)
	if err != nil {
		log.Error().Err(err).Str("user_id", userUUID).Msg("failed to get user urls")
		return nil, fmt.Errorf("failed to get user urls: %w", err)
	}

	return urls, nil
}

// DeleteURLs асинхронно удаляет URL пользователя.
func (s *Service) DeleteURLs(ctx context.Context, shortURLs []string, userUUID string) error {
	if userUUID == "" {
		return model.ErrUnauthenticated
	}
	return s.storage.DeleteURLs(ctx, shortURLs, userUUID)
}

// ShortenBatch выполняет пакетное сокращение URL.
func (s *Service) ShortenBatch(ctx context.Context, items []model.BatchRequestItem, userUUID string) ([]model.BatchResponseItem, error) {
	if userUUID == "" {
		return nil, model.ErrUnauthenticated
	}
	for _, item := range items {
		if !isValidURL(item.LongURL) {
			return nil, model.ErrInvalidURL
		}
	}
	return s.storage.GetShortList(ctx, items, userUUID)
}

// StatsResult содержит статистику сервиса.
type StatsResult struct {
	URLs  int
	Users int
}

// Stats возвращает количество URL и уникальных пользователей.
func (s *Service) Stats(ctx context.Context) (*StatsResult, error) {
	urls, users, err := s.storage.Stats(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get stats")
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	return &StatsResult{URLs: urls, Users: users}, nil
}

// isValidURL проверяет, является ли строка корректным HTTP или HTTPS URL.
func isValidURL(rawURL string) bool {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return false
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return u.Host != ""
}
