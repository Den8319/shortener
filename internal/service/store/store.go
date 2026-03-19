package store

import (
	"sync"
	"fmt"

	"github.com/Den8319/shortener/pkg/generator"
)


type Store struct {
	urls map[string]string // map[shortURL]longURL
	mu   sync.RWMutex
	

}

const shortURLLength = 8


type Storage interface {
    GetShortURL(longURL string) (string, error)
    GetLongURL(shortURL string) (string, error)
}

func New() *Store {
	return &Store{urls: make(map[string]string)}
}


// GetShortURL возвращает существующий short для longURL или создаёт и сохраняет новый.
func (s *Store) GetShortURL(longURL string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for short, storedLong := range s.urls {
		if storedLong == longURL {
			return short, nil
		}
	}

	short, err := generator.GenerateShort(shortURLLength)
	if err != nil {
		return "", err
	}
	s.urls[short] = longURL
	return short, nil
}

func (s *Store)  GetLongURL(shortURL string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	longURL, exists := s.urls[shortURL]
	if !exists {
		return "", fmt.Errorf("short URL not found")
	}
	return longURL, nil
}