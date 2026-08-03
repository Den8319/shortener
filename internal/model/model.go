package model

// Константы для работы с куками и контекстом запроса.
const (
	// CookieAuth — имя куки для JWT-токена аутентификации.
	CookieAuth = "Auth"
	// CookieUser — имя куки для открытого идентификатора пользователя.
	CookieUser = "User"

	// ContextValueUser — ключ для хранения идентификатора пользователя в контексте запроса.
	ContextValueUser = "User"
)

// generate:reset
// URL представляет собой пару короткого и длинного URL, ассоциированную с пользователем.
type URL struct {
	ShortURL  string `json:"short_url"`
	LongURL   string `json:"original_url"`
	UserUUID  string `json:"user_id,omitempty"`
	IsDeleted bool   `json:"is_deleted,omitempty"`
}

// generate:reset
// Request — тело запроса на сокращение URL в формате JSON.
type Request struct {
	LongURL string `json:"url"`
}

// generate:reset
// Response — тело ответа на сокращение URL в формате JSON.
type Response struct {
	ShortURL string `json:"result"`
}

// generate:reset
// BatchRequestItem — элемент пакетного запроса на сокращение URL.
type BatchRequestItem struct {
	Corr    string `json:"correlation_id"`
	LongURL string `json:"original_url"`
}

// generate:reset
// BatchResponseItem — элемент пакетного ответа с результатами сокращения URL.
type BatchResponseItem struct {
	Corr     string `json:"correlation_id"`
	ShortURL string `json:"short_url"`
}
