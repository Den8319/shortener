package file

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Den8319/shortener/internal/model"
	"github.com/google/uuid"
)

type record struct {
	UUID     uuid.UUID `json:"uuid"`
	ShortURL string    `json:"short_url"`
	LongURL  string    `json:"long_url"`
}

type Fileloader struct {
	filePath string
	file     *os.File
	encoder  *json.Encoder
}

var _ model.Loader = (*Fileloader)(nil)

func New(filePath string) (*Fileloader, error) {
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	return &Fileloader{
		filePath: filePath,
		file:     f,
		encoder:  json.NewEncoder(f),
	}, nil
}

func (r *Fileloader) Load(ctx context.Context) (map[string]string, error) {
	f, err := os.OpenFile(r.filePath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("open file for read: %w", err)
	}
	defer f.Close()

	urls := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var rec record
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			return nil, fmt.Errorf("parse record: %w", err)
		}
		urls[rec.ShortURL] = rec.LongURL
	}
	return urls, scanner.Err()
}

func (r *Fileloader) Save(ctx context.Context, url *model.URL) error {
	return r.encoder.Encode(record{
		UUID:     uuid.New(),
		ShortURL: url.ShortURL,
		LongURL:  url.LongURL,
	})
}

func (r *Fileloader) GetLongURL(ctx context.Context, shortURL string) (string, error) {
	f, err := os.OpenFile(r.filePath, os.O_RDONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("open file for read: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var rec record
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}
		if rec.ShortURL == shortURL {
			return rec.LongURL, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scan file: %w", err)
	}

	return "", fmt.Errorf("url not found")
}

func (r *Fileloader) Close() error {
	return r.file.Close()
}
