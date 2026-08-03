package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// AuditEvent представляет собой событие аудита.
type AuditEvent struct {
	Ts     int64  `json:"ts"`      // Unix timestamp события
	Action string `json:"action"`  // Действие: "shorten" или "follow"
	UserID string `json:"user_id"` // Идентификатор пользователя
	URL    string `json:"url"`     // Оригинальный URL
}

// Observer — интерфейс наблюдателя для получения событий аудита.
type Observer interface {
	Notify(event AuditEvent) error
}

// Closer — интерфейс для освобождения ресурсов наблюдателя.
type Closer interface {
	Close() error
}

// FileObserver записывает события аудита в файл в формате JSON.
type FileObserver struct {
	file    *os.File
	encoder *json.Encoder
	mu      sync.Mutex
}

// NewFileObserver создаёт FileObserver, открывая файл для добавления записей.
// Если файл не существует, он будет создан.
func NewFileObserver(filePath string) (*FileObserver, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit file: %w", err)
	}
	return &FileObserver{
		file:    file,
		encoder: json.NewEncoder(file),
	}, nil
}

// Notify записывает событие аудита в файл в формате JSON.
func (fo *FileObserver) Notify(event AuditEvent) error {
	fo.mu.Lock()
	defer fo.mu.Unlock()

	err := fo.encoder.Encode(event)
	if err != nil {
		return fmt.Errorf("failed to encode and write audit event: %w", err)
	}
	return nil
}

// Close закрывает файл аудита.
func (fo *FileObserver) Close() error {
	if fo.file != nil {
		return fo.file.Close()
	}
	return nil
}

// HTTPObserver отправляет события аудита на удалённый HTTP-эндпоинт.
type HTTPObserver struct {
	url    string
	client *http.Client
}

// NewHTTPObserver создаёт HTTPObserver с таймаутом 3 секунды.
func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url: url,
		client: &http.Client{
			Timeout: 3 * time.Second, // Защита от зависания сети
		},
	}
}

// Notify отправляет событие аудита POST-запросом в формате JSON.
func (ho *HTTPObserver) Notify(event AuditEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("http observer marshal error: %w", err)
	}

	resp, err := ho.client.Post(ho.url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("http observer request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http observer got unexpected status: %d", resp.StatusCode)
	}
	return nil
}

// Auditor управляет коллекцией наблюдателей и рассылает им события аудита.
type Auditor struct {
	observers []Observer
	mu        sync.RWMutex
}

// NewAuditor создаёт новый Auditor без наблюдателей.
func NewAuditor() *Auditor {
	return &Auditor{
		observers: make([]Observer, 0),
	}
}

// Register добавляет наблюдателя в список получателей событий аудита.
func (a *Auditor) Register(observer Observer) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.observers = append(a.observers, observer)
}

// Close закрывает всех наблюдателей, реализующих интерфейс Closer.
// Ошибки закрытия логируются, но не прерывают цикл — закрываются все наблюдатели.
func (a *Auditor) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, observer := range a.observers {
		if closer, ok := observer.(Closer); ok {
			if err := closer.Close(); err != nil {
				log.Error().Err(err).Msg("[Audit Error] failed to close observer")
			}
		}
	}
	return nil
}

// Notify отправляет событие аудита всем зарегистрированным наблюдателям.
// Ошибки отдельных наблюдателей логируются, но не блокируют отправку остальным.
func (a *Auditor) Notify(event AuditEvent) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, observer := range a.observers {
		if err := observer.Notify(event); err != nil {
			log.Error().Err(err).Msg("[Audit Error] failed to notify observer")

		}
	}
}
