package config

import (
	"flag"
	"os"
)

const (
	envServerAddress     = "SERVER_ADDRESS"
	flagServerAddress    = "a"
	defaultServerAddress = ":8080"

	envBaseAddress     = "BASE_URL"
	flagBaseAddress    = "b"
	defaultBaseAddress = "http://localhost:8080"
)

type Config struct {
	ServerAddress string
	BaseURL       string // Например:  "http://localhost:8080"
}

func getAddress(envName, flagValue string) string {

	if envValue, exists := os.LookupEnv(envName); exists && envValue != "" {
		return envValue
	}

	return flagValue
}

func New() *Config {

	serverAddr := flag.String(flagServerAddress, defaultServerAddress, "")
	baseURL := flag.String(flagBaseAddress, defaultBaseAddress, "")

	flag.Parse()

	cfg := &Config{
		ServerAddress: getAddress(envServerAddress, *serverAddr),
		BaseURL:       getAddress(envBaseAddress, *baseURL),
	}

	return cfg
}
