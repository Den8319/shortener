// Package file реализует хранилище URL в JSON-файле с кэшированием в памяти.
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

// Fileloader реализует интерфейс model.Loader для хранения данных в JSON-файле.
// Поддерживает кэширование записей в памяти с TTL 5 секунд.
type Fileloader struct {
	filePath    string
	file        *os.File
	encoder     *json.Encoder
	cache       *CachedRecord
	cacheMu     sync.RWMutex
	lastModTime time.Time
}

var _ model.Loader = (*Fileloader)(nil)

// New создаёт новый Fileloader, открывая файл для добавления записей.
// Если файл не существует, он будет создан.
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

// Load читает все записи из файла и возвращает карту соответствия коротких URL длинным.
// Использует кэш с TTL 5 секунд для ускорения повторных чтений.
// Записи с флагом IsDeleted=true исключаются из результата.
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

// Save записывает пару короткий/длинный URL в файл в формате JSON.
// После записи сбрасывает кэш, чтобы при следующем чтении данные были актуальны.
func (r *Fileloader) Save(ctx context.Context, url *model.URL, userUUID string) error {
	if err := r.encoder.Encode(record{
		UUID:      uuid.New(),
		ShortURL:  url.ShortURL,
		LongURL:   url.LongURL,
		UserUUID:  userUUID,
		IsDeleted: false,
	}); err != nil {
		return fmt.Errorf("failed to encode record: %w", err)
	}

	// Сбрасываем кэш, чтобы последующие чтения подтянули новые данные.
	r.cacheMu.Lock()
	r.cache = nil
	r.cacheMu.Unlock()

	return nil
}

// GetLongURL возвращает длинный URL по короткому идентификатору.
// Сначала проверяет кэш, при промахе читает файл. Возвращает ErrURLDeleted для удалённых записей.
func (r *Fileloader) GetLongURL(ctx context.Context, shortURL string) (string, error) {
	// Проверяем кэш
	r.cacheMu.RLock()
	cache := r.cache
	if cache != nil && time.Since(cache.cacheTime) < cacheTTL {
		idx, exists := cache.shortURLIndex[shortURL]
		r.cacheMu.RUnlock()
		if exists && idx >= 0 && idx < len(cache.records) {
			rec := cache.records[idx]
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

// GetUserURLs возвращает все URL, созданные указанным пользователем.
// Использует кэш с индексом по userUUID для ускорения поиска.
// Удалённые записи (IsDeleted=true) исключаются из результата.
func (r *Fileloader) GetUserURLs(ctx context.Context, userUUID string) ([]model.URL, error) {
	// Проверяем кэш
	r.cacheMu.RLock()
	cache := r.cache
	if cache != nil && time.Since(cache.cacheTime) < cacheTTL {
		indices, exists := cache.userIndex[userUUID]
		r.cacheMu.RUnlock()
		if exists {
			var urls []model.URL
			for _, idx := range indices {
				if idx >= 0 && idx < len(cache.records) {
					rec := cache.records[idx]
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

// Close закрывает файл, открытый для записи, с предварительным сбросом данных на диск.
func (r *Fileloader) Close() error {
	if err := r.file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}
	return r.file.Close()
}

// Delete не поддерживается в файловом хранилище и всегда возвращает ошибку.
func (r *Fileloader) Delete(ctx context.Context, shortURLs []string, userUUID string) error {
	return errors.New("unsupport")
}

// Stats не поддерживается в файловом хранилище и всегда возвращает ошибку.
func (r *Fileloader) Stats(ctx context.Context) (int, int, error) {
	return 0, 0, errors.New("stats not supported for file storage")
}
