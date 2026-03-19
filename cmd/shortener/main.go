package main

import (
	"github.com/Den8319/shortener/internal/handler"
	"github.com/Den8319/shortener/internal/config"
	"github.com/Den8319/shortener/internal/service/store"
	"net/http"

	"github.com/go-chi/chi/v5"

)

func main() {
	cfg := config.New()
	route := chi.NewRouter()
	s := store.New()
	h := handler.NewHandler(s,cfg.BaseURL)

	route.Post("/", h.HandPostFullURL) 
	route.Get("/{id}", h.HandGetURL) 

	
	err := http.ListenAndServe(cfg.ServerAddress, route)
	if err != nil {
		panic(err)



	}
}
