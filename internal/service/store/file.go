package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type record struct {
	ShortURL string `json:"short_url"`
	LongURL  string `json:"long_url"`
}

 
type FileStore struct {
	*Store
	filePath string
}

 
func NewFileStore(filePath string) (*FileStore, error) {
	fs := &FileStore{
		Store:    New(),
		filePath: filePath,
	}
	if err := fs.load(); err != nil {
		return nil, err
	}
	return fs, nil
}
 
func (fs *FileStore) load() error {
	f, err := os.OpenFile(fs.filePath, os.O_RDONLY|os.O_CREATE, 0644)
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
		fs.Store.add(rec.ShortURL, rec.LongURL)
	}
	return scanner.Err()
}

// appendRecord дописывает одну запись в конец файла.
func (fs *FileStore) appendRecord(rec record) error {
	f, err := os.OpenFile(fs.filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open storage file for append: %w", err)
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(rec)
}

// GetShortURL возвращает короткий URL для переданного longURL.
func (fs *FileStore) GetShortURL(longURL string) (string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	short, isNew, err := fs.findOrGenerate(longURL)
	if err != nil || !isNew {
		return short, err
	}

	if err := fs.appendRecord(record{ShortURL: short, LongURL: longURL}); err != nil {
		return "", err
	}

	fs.add(short, longURL)
	return short, nil
}
