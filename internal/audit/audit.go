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

// AuditEvent представляет собой событие аудита
type AuditEvent struct {
	Ts     int64  `json:"ts"`      // Unix timestamp события
	Action string `json:"action"`  // Действие: "shorten" или "follow"
	UserID string `json:"user_id"` // Идентификатор пользователя
	URL    string `json:"url"`     // Оригинальный URL
}

// Observer интерфейс наблюдателя для получения событий аудита
type Observer interface {
	Notify(event AuditEvent) error
}


type FileObserver struct {
	file *os.File
	mu   sync.Mutex
}

func NewFileObserver(filePath string) (*FileObserver, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit file: %w", err)
	}
	return &FileObserver{file: file}, nil
}

func (fo *FileObserver) Notify(event AuditEvent) error {
	fo.mu.Lock()
	defer fo.mu.Unlock()

	err := json.NewEncoder(fo.file).Encode(event)
	if err != nil {
		return fmt.Errorf("failed to encode and write audit event: %w", err)
	}
	return nil
}

func (fo *FileObserver) Close() error {
	if fo.file != nil {
		return fo.file.Close()
	}
	return nil
}

type HTTPObserver struct {
	url    string
	client *http.Client
}

func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url: url,
		client: &http.Client{
			Timeout: 3 * time.Second, // Защита от зависания сети
		},
	}
}

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


type Auditor struct {
	observers []Observer
	mu        sync.RWMutex
}

func NewAuditor() *Auditor {
	return &Auditor{
		observers: make([]Observer, 0),
	}
}


func (a *Auditor) Register(observer Observer) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.observers = append(a.observers, observer)
}


func (a *Auditor) Notify(event AuditEvent) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, observer := range a.observers {
		if err := observer.Notify(event); err != nil {
			log.Error().Err(err).Msg("[Audit Error] failed to notify observer")
			
		}
	}
}
