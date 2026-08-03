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

// URL представляет собой пару короткого и длинного URL, ассоциированную с пользователем.
// generate:reset
type URL struct {
	ShortURL  string `json:"short_url"`
	LongURL   string `json:"original_url"`
	UserUUID  string `json:"user_id,omitempty"`
	IsDeleted bool   `json:"is_deleted,omitempty"`
}

// Request — тело запроса на сокращение URL в формате JSON.
// generate:reset
type Request struct {
	LongURL string `json:"url"`
}

// Response — тело ответа на сокращение URL в формате JSON.
// generate:reset
type Response struct {
	ShortURL string `json:"result"`
}

// BatchRequestItem — элемент пакетного запроса на сокращение URL.
// generate:reset
type BatchRequestItem struct {
	Corr    string `json:"correlation_id"`
	LongURL string `json:"original_url"`
}

// BatchResponseItem — элемент пакетного ответа с результатами сокращения URL.
// generate:reset
type BatchResponseItem struct {
	Corr     string `json:"correlation_id"`
	ShortURL string `json:"short_url"`
}
