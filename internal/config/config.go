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

	envLogLevel     = "LOG_LEVEL"
	flagLogLevel    = "l"	
	defaultLogLevel = "Info"
)

type Config struct {
	ServerAddress string
	BaseURL       string // Например:  "http://localhost:8080"
	LogLevel      string
}

func getParam(envName, flagValue string) string {

	if envValue, exists := os.LookupEnv(envName); exists && envValue != "" {
		return envValue
	}

	return flagValue
}

func New() *Config {

	serverAddr := flag.String(flagServerAddress, defaultServerAddress, "")
	baseURL := flag.String(flagBaseAddress, defaultBaseAddress, "")
	logLevel := flag.String(flagLogLevel, defaultLogLevel, "")

	flag.Parse()

	cfg := &Config{
		ServerAddress: getParam(envServerAddress, *serverAddr),
		BaseURL:       getParam(envBaseAddress, *baseURL),
		LogLevel:      getParam(envLogLevel, *logLevel),
	}

	return cfg
}







