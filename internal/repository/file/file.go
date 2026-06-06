package file

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/Den8319/shortener/internal/model"
	"github.com/google/uuid"
)

type record struct {
	UUID      uuid.UUID `json:"uuid"`
	ShortURL  string    `json:"short_url"`
	LongURL   string    `json:"long_url"`
	UserUUID  string    `json:"user_uuid"`
	IsDeleted bool      `json:"is_deleted"`
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
		if !rec.IsDeleted {
			urls[rec.ShortURL] = rec.LongURL
		}
	}
	return urls, scanner.Err()
}

func (r *Fileloader) Save(ctx context.Context, url *model.URL, userUUID string) error {
	return r.encoder.Encode(record{
		UUID:      uuid.New(),
		ShortURL:  url.ShortURL,
		LongURL:   url.LongURL,
		UserUUID:  userUUID,
		IsDeleted: false,
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
			if rec.IsDeleted {
				return "", model.ErrURLDeleted
			}
			return rec.LongURL, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scan file: %w", err)
	}

	return "", fmt.Errorf("url not found")
}

func (r *Fileloader) GetUserURLs(ctx context.Context, userUUID string) ([]model.URL, error) {
	f, err := os.OpenFile(r.filePath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open file for read: %w", err)
	}
	defer f.Close()

	var urls []model.URL
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var rec record
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}
		if rec.UserUUID == userUUID && !rec.IsDeleted {
			urls = append(urls, model.URL{ShortURL: rec.ShortURL, LongURL: rec.LongURL})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan file: %w", err)
	}

	return urls, nil
}

func (r *Fileloader) Close() error {
	return r.file.Close()
}

func (r *Fileloader) Delete(ctx context.Context, shortURLs []string, userUUID string) error {
	 
	return 	errors.New("unsupport")
}

func (r *Fileloader) findShortByLongURL(longURL string) (string, error) {
	f, err := os.OpenFile(r.filePath, os.O_RDONLY, 0644)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var rec record
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}
		if rec.LongURL == longURL {
			return rec.ShortURL, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", nil
}
