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

	envDatabaseDSN     = "DATABASE_DSN"
	flagDatabaseDSN    = "d"
	defaultDatabaseDSN = ""

	envSecretKey     = "SECRET_KEY"
	flagSecretKey    = "k"
	defaultSecretKey = ""
)

type Config struct {
	ServerAddress   string
	BaseURL         string
	LogLevel        string
	FileStoragePath string
	DatabaseDSN     string
	SecretKey       string
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
	databaseDSN := flag.String(flagDatabaseDSN, defaultDatabaseDSN, "")
	secretKey := flag.String(flagSecretKey, defaultSecretKey, "")

	flag.Parse()

	cfg := &Config{
		ServerAddress:   getParam(envServerAddress, *serverAddr),
		BaseURL:         getParam(envBaseAddress, *baseURL),
		LogLevel:        getParam(envLogLevel, *logLevel),
		FileStoragePath: getParam(envFileStoragePath, *filePath),
		DatabaseDSN:     getParam(envDatabaseDSN, *databaseDSN),
		SecretKey:       getParam(envSecretKey, *secretKey),
	}

	return cfg
}
