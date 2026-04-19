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

	envFileStoragePath     = "FILE_STORAGE_PATH"
	flagFileStoragePath    = "f"
	defaultFileStoragePath = "./short-url-db.json"
)

type Config struct {
	ServerAddress   string
	BaseURL         string // Например:  "http://localhost:8080"
	LogLevel        string
	FileStoragePath string
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
	filePath := flag.String(flagFileStoragePath, defaultFileStoragePath, "")

	flag.Parse()

	cfg := &Config{
		ServerAddress:   getParam(envServerAddress, *serverAddr),
		BaseURL:         getParam(envBaseAddress, *baseURL),
		LogLevel:        getParam(envLogLevel, *logLevel),
		FileStoragePath: getParam(envFileStoragePath, *filePath),
	}

	return cfg
}
