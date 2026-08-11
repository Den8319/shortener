// Package model содержит типы данных, интерфейсы и ошибки, используемые в проекте.
package model

import "errors"

// Доменные ошибки сервиса. Обработчики отображают их в транспортные коды
// (HTTP status codes / gRPC codes), не зависящие от внутренней реализации.
var (
	// ErrURLAlreadyExists возвращается при попытке создать уже существующий короткий URL.
	ErrURLAlreadyExists = errors.New("already exists")

	// ErrURLDeleted возвращается при попытке получить URL, помеченный как удалённый.
	ErrURLDeleted = errors.New("URL has been deleted")

	// ErrInvalidURL возвращается при некорректном URL.
	ErrInvalidURL = errors.New("invalid url")

	// ErrEmptyID возвращается при пустом идентификаторе короткого URL.
	ErrEmptyID = errors.New("empty id")

	// ErrURLNotFound возвращается, когда короткий URL не найден.
	ErrURLNotFound = errors.New("url not found")

	// ErrUnauthenticated возвращается при отсутствии или некорректной аутентификации пользователя.
	ErrUnauthenticated = errors.New("unauthenticated")
)
