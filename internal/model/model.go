package model

const (
	CookieAuth = "Auth"
	CookieUser = "User"

	ContextValueUser = "User"
)

type URL struct {
	ShortURL string `json:"short_url"`
	LongURL  string `json:"original_url"`
}

type Request struct {
	LongURL string `json:"url"`
}

type Response struct {
	ShortURL string `json:"result"`
}

type BatchRequestItem struct {
	Corr    string `json:"correlation_id"`
	LongURL string `json:"original_url"`
}

type BatchResponseItem struct {
	Corr     string `json:"correlation_id"`
	ShortURL string `json:"short_url"`
}
