package main

import (
	"github.com/Den8319/shortener/internal/handler"
	"github.com/Den8319/shortener/internal/service/store"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	s := store.New()
	h := handler.NewHandler(s)

	

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/" {
			h.HandPostFullURL(w, r)
			return
		}

		if r.Method == http.MethodGet && r.URL.Path != "/" {
			h.HandGetURL(w, r)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	})

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
