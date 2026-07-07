package file

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Den8319/shortener/internal/model"
	"github.com/google/uuid"
)

const (
	defaultBufferSize = 1024 * 1024     // 1 MB буфер для сканера
	cacheTTL          = 5 * time.Second // время жизни кэша
)

type record struct {
	UUID      uuid.UUID `json:"uuid"`
	ShortURL  string    `json:"short_url"`
	LongURL   string    `json:"long_url"`
	UserUUID  string    `json:"user_uuid"`
	IsDeleted bool      `json:"is_deleted"`
}

// CachedRecord — структура для кэшированных данных
type CachedRecord struct {
	records       []record
	cacheTime     time.Time
	fileModTime   time.Time
	userIndex     map[string][]int // индекс userUUID → индексы в records
	shortURLIndex map[string]int   // индекс shortURL → индекс в records
}

type Fileloader struct {
	filePath    string
	file        *os.File
	encoder     *json.Encoder
	cache       *CachedRecord
	cacheMu     sync.RWMutex
	lastModTime time.Time
}

var _ model.Loader = (*Fileloader)(nil)

func New(filePath string) (*Fileloader, error) {
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	return &Fileloader{
		filePath:    filePath,
		file:        f,
		encoder:     json.NewEncoder(f),
		lastModTime: time.Now(),
	}, nil
}

func (r *Fileloader) Load(ctx context.Context) (map[string]string, error) {
	// Проверяем кэш
	r.cacheMu.RLock()
	if r.cache != nil && time.Since(r.cache.cacheTime) < cacheTTL {
		defer r.cacheMu.RUnlock()
		urls := make(map[string]string)
		for _, rec := range r.cache.records {
			if !rec.IsDeleted {
				urls[rec.ShortURL] = rec.LongURL
			}
		}
		return urls, nil
	}
	r.cacheMu.RUnlock()

	f, err := os.OpenFile(r.filePath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("open file for read: %w", err)
	}
	defer f.Close()

	// Получаем время модификации файла
	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	// Создаем сканер с увеличенным буфером
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, defaultBufferSize)
	scanner.Buffer(buf, defaultBufferSize*2)

	urls := make(map[string]string)
	var records []record
	userIndex := make(map[string][]int)
	shortURLIndex := make(map[string]int)

	for scanner.Scan() {
		var rec record
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			return nil, fmt.Errorf("parse record: %w", err)
		}
		if !rec.IsDeleted {
			urls[rec.ShortURL] = rec.LongURL
		}
		records = append(records, rec)
		userIndex[rec.UserUUID] = append(userIndex[rec.UserUUID], len(records)-1)
		shortURLIndex[rec.ShortURL] = len(records) - 1
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan file: %w", err)
	}

	// Обновляем кэш
	r.cacheMu.Lock()
	r.cache = &CachedRecord{
		records:       records,
		cacheTime:     time.Now(),
		fileModTime:   stat.ModTime(),
		userIndex:     userIndex,
		shortURLIndex: shortURLIndex,
	}
	r.cacheMu.Unlock()

	return urls, nil
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
	// Проверяем кэш
	r.cacheMu.RLock()
	if r.cache != nil && time.Since(r.cache.cacheTime) < cacheTTL {
		idx, exists := r.cache.shortURLIndex[shortURL]
		r.cacheMu.RUnlock()
		if exists && idx >= 0 && idx < len(r.cache.records) {
			rec := r.cache.records[idx]
			if rec.ShortURL == shortURL {
				if rec.IsDeleted {
					return "", model.ErrURLDeleted
				}
				return rec.LongURL, nil
			}
		}
	} else {
		r.cacheMu.RUnlock()
	}

	// Кэш не найден или устарел, читаем с диска
	f, err := os.OpenFile(r.filePath, os.O_RDONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("open file for read: %w", err)
	}
	defer f.Close()

	// Создаем сканер с увеличенным буфером
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, defaultBufferSize)
	scanner.Buffer(buf, defaultBufferSize*2)

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
	// Проверяем кэш
	r.cacheMu.RLock()
	if r.cache != nil && time.Since(r.cache.cacheTime) < cacheTTL {
		indices, exists := r.cache.userIndex[userUUID]
		r.cacheMu.RUnlock()
		if exists {
			var urls []model.URL
			for _, idx := range indices {
				if idx >= 0 && idx < len(r.cache.records) {
					rec := r.cache.records[idx]
					if !rec.IsDeleted {
						urls = append(urls, model.URL{ShortURL: rec.ShortURL, LongURL: rec.LongURL})
					}
				}
			}
			return urls, nil
		}
	} else {
		r.cacheMu.RUnlock()
	}

	// Кэш не найден или устарел, читаем с диска
	f, err := os.OpenFile(r.filePath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open file for read: %w", err)
	}
	defer f.Close()

	// Создаем сканер с увеличенным буфером
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, defaultBufferSize)
	scanner.Buffer(buf, defaultBufferSize*2)

	var urls []model.URL
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
	return errors.New("unsupport")
}
