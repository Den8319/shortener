package config

import (
    "flag"
	
)

//Флаг -a  отвечает за адрес запуска HTTP-сервера (значение может быть таким: localhost:8888).
//Флаг -b отвечает за базовый адрес результирующего сокращённого URL (значение: адрес сервера перед коротким URL, например, http://localhost:8000/qsd54gFg).

type Config struct {
	ServerAddress  string
	BaseURL        string
}


func New() *Config{
    
	cfg := &Config{}

	flag.StringVar(&cfg.ServerAddress, "a", ":8080", "Порт HTTP сервера")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8000", "Базовый адрес для коротких URL")


	

    flag.Parse()


	return cfg
}



