package model

type Request struct {
	LongUrl string `json:"url"`
}

type Response struct {
	ShortUrl string `json:"result"`
}
