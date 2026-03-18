package main

import (
	"github.com/Den8319/shortener/internal/handler"
	"github.com/Den8319/shortener/internal/service/store"
	"net/http"

	"github.com/go-chi/chi/v5"

)

func main() {
	route := chi.NewRouter()
	s := store.New()
	h := handler.NewHandler(s)

	route.Post("/", h.HandPostFullURL) 
	route.Get("/{id}", h.HandGetURL) 

	
	err := http.ListenAndServe(`:8080`, route)
	if err != nil {
		panic(err)



	}
}
