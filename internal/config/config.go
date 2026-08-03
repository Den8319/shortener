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

	envAuditFile     = "AUDIT_FILE"
	flagAuditFile    = "audit-file"
	defaultAuditFile = ""

	envAuditURL     = "AUDIT_URL"
	flagAuditURL    = "audit-url"
	defaultAuditURL = ""
)

// Config содержит все параметры конфигурации приложения.
// Значения читаются из переменных окружения, а при их отсутствии — из флагов командной строки.
// Если не задано ни то, ни другое, используются значения по умолчанию.
type Config struct {
	ServerAddress   string
	BaseURL         string
	LogLevel        string
	FileStoragePath string
	DatabaseDSN     string
	SecretKey       string
	AuditFile       string
	AuditURL        string
}

// New создаёт Config, читая флаги из os.Args[1:].
func New() *Config {

	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	return NewWithFlagSet(fs, os.Args[1:])
}

// NewWithFlagSet создаёт Config из указанного FlagSet и аргументов.
// Используется в тестах для передачи произвольных флагов.
func NewWithFlagSet(fs *flag.FlagSet, args []string) *Config {
	serverAddr := fs.String(flagServerAddress, defaultServerAddress, "")
	baseURL := fs.String(flagBaseAddress, defaultBaseAddress, "")
	logLevel := fs.String(flagLogLevel, defaultLogLevel, "")
	filePath := fs.String(flagFileStoragePath, defaultFileStoragePath, "")
	databaseDSN := fs.String(flagDatabaseDSN, defaultDatabaseDSN, "")
	secretKey := fs.String(flagSecretKey, defaultSecretKey, "")
	auditFile := fs.String(flagAuditFile, defaultAuditFile, "")
	auditURL := fs.String(flagAuditURL, defaultAuditURL, "")

	// Игнорируем ошибку парсинга, чтобы не падать на неизвестных флагах
	_ = fs.Parse(args)

	return &Config{
		ServerAddress:   getParam(envServerAddress, *serverAddr),
		BaseURL:         getParam(envBaseAddress, *baseURL),
		LogLevel:        getParam(envLogLevel, *logLevel),
		FileStoragePath: getParam(envFileStoragePath, *filePath),
		DatabaseDSN:     getParam(envDatabaseDSN, *databaseDSN),
		SecretKey:       getParam(envSecretKey, *secretKey),
		AuditFile:       getParam(envAuditFile, *auditFile),
		AuditURL:        getParam(envAuditURL, *auditURL),
	}
}

// getParam возвращает значение из ENV, если оно задано, иначе из флагов.
func getParam(envName, flagValue string) string {
	if envValue, exists := os.LookupEnv(envName); exists && envValue != "" {
		return envValue
	}
	return flagValue
}
