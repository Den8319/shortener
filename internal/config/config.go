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
	BaseURL       string // Например: "http://localhost:8080"
}

func setParam(envName, flagName, defaultValue string) string {

	if envValue, exists := os.LookupEnv(envName); exists && envValue != "" {
		return envValue
	}

	flagAddress := flag.String(flagName, defaultValue, "")

	return *flagAddress
}

func New() *Config {

	flag.Parse()

	cfg := &Config{
		ServerAddress: setParam(envServerAddress, flagServerAddress, defaultServerAddress),
		BaseURL:       setParam(envBaseAddress, flagBaseAddress, defaultBaseAddress),
	}

	return cfg
}
