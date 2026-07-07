package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	file "github.com/Den8319/shortener/internal/repository/file"
)

// ExampleNew демонстрирует создание Store и сокращение URL
func ExampleNew() {
	tmpDir, _ := os.MkdirTemp("", "store-example-")
	defer os.RemoveAll(tmpDir)

	loader, _ := file.New(filepath.Join(tmpDir, "store.json"))
	store, _ := New(loader)
	defer store.Close()

	shortURL, _ := store.GetShortURL(context.Background(), "https://example.com", "user-123")
	fmt.Println("Длина:", len(shortURL))
	// Output:
	// Длина: 8
}

// ExampleStore_GetLongURL демонстрирует получение длинного URL
func ExampleStore_GetLongURL() {
	tmpDir, _ := os.MkdirTemp("", "store-example-")
	defer os.RemoveAll(tmpDir)

	loader, _ := file.New(filepath.Join(tmpDir, "store.json"))
	store, _ := New(loader)
	defer store.Close()

	longURL := "https://my-website.com"
	shortURL, _ := store.GetShortURL(context.Background(), longURL, "user-456")
	retrievedURL, _ := store.GetLongURL(context.Background(), shortURL)

	fmt.Println("Совпадение:", longURL == retrievedURL)
	// Output:
	// Совпадение: true
}