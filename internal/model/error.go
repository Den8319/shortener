// Package model содержит типы данных, интерфейсы и ошибки, используемые в проекте.
package model

import "errors"

// ErrURLAlreadyExists возвращается при попытке создать уже существующий короткий URL.
var ErrURLAlreadyExists = errors.New("already exists")

// ErrURLDeleted возвращается при попытке получить URL, помеченный как удалённый.
var ErrURLDeleted = errors.New("URL has been deleted")
