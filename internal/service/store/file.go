package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
)

type record struct {
	UUID     uuid.UUID `json:"uuid"`
	ShortURL string    `json:"short_url"`
	LongURL  string    `json:"long_url"`
}

type FileStore struct {
	*Store
	file    *os.File
	encoder *json.Encoder
}

func NewFileStore(filePath string) (*FileStore, error) {
	store := New()

	if err := loadFromFile(filePath, store); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open storage file for write: %w", err)
	}

	return &FileStore{
		Store:   store,
		file:    f,
		encoder: json.NewEncoder(f),
	}, nil
}

func loadFromFile(filePath string, s *Store) error {
	f, err := os.OpenFile(filePath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("open storage file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var rec record
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			return fmt.Errorf("parse record: %w", err)
		}
		s.add(rec.ShortURL, rec.LongURL)
	}
	return scanner.Err()
}

func (fs *FileStore) appendRecord(rec record) error {
	return fs.encoder.Encode(rec)
}

func (fs *FileStore) Close() error {
	return fs.file.Close()
}

// GetShortURL возвращает короткий URL для переданного longURL.
func (fs *FileStore) GetShortURL(longURL string) (string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	short, isNew, err := fs.findOrGenerate(longURL)
	if err != nil || !isNew {
		return short, err
	}

	if err := fs.appendRecord(record{UUID: uuid.New(), ShortURL: short, LongURL: longURL}); err != nil {
		return "", err
	}

	fs.add(short, longURL)
	return short, nil
}
