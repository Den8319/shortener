package model

type Request struct {
	LongURL string `json:"url"`
}

type Response struct {
	ShortURL string `json:"result"`
}
