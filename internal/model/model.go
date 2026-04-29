package model

type URL struct {
	ShortURL string `json:"short_url"`
	LongURL  string `json:"long_url"`
}

type Request struct {
	LongURL string `json:"url"`
}

type Response struct {
	ShortURL string `json:"result"`
}
