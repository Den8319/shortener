package store

import (
	"fmt"
	"sync"

	"github.com/Den8319/shortener/pkg/generator"
)

const shortURLLength = 8

type Store struct {
	urls   map[string]string
	mu     sync.RWMutex
}

func New() *Store {
	return &Store{
		urls: make(map[string]string),	
	}
}


func (s *Store) add(short, long string) {
	s.urls[short] = long
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

func (s *Store) GetLongURL(shortURL string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	long, ok := s.urls[shortURL]
	if !ok {
		return "", fmt.Errorf("short URL not found")
	}
	return long, nil
}
