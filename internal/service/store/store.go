package store

import (
	"fmt"
	"sync"

	"github.com/Den8319/shortener/pkg/generator"
)

const shortURLLength = 8

type Loader interface {
	Load() (map[string]string, error)
	Save(short, long string) error
	Close() error
}

type Store struct {
	urls   map[string]string
	mu     sync.RWMutex
	loader Loader
}

func New(loader Loader) (*Store, error) {
	s := &Store{
		urls:   make(map[string]string),
		loader: loader,
	}

	if loader != nil {
		urls, err := loader.Load()
		if err != nil {
			return nil, err
		}
		s.urls = urls
	}

	return s, nil
}

func (s *Store) GetShortURL(longURL string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	short, isNew, err := s.findOrGenerate(longURL)
	if err != nil || !isNew {
		return short, err
	}

	s.urls[short] = longURL

	if s.loader != nil {
		s.loader.Save(short, longURL)
	}

	return short, nil
}

func (s *Store) GetLongURL(shortURL string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	long, ok := s.urls[shortURL]
	if !ok {
		return "", fmt.Errorf("short URL not found")
	}
	return long, nil
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
