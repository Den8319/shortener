// Package config реализует загрузку конфигурации приложения из JSON-файла,
// переменных окружения и флагов командной строки.
//
// Приоритет значений (от высшего к низшему):
//  1. Переменные окружения
//  2. Флаги командной строки (если заданы явно)
//  3. JSON-файл конфигурации
//  4. Значения по умолчанию
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
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

	envEnableHTTPS     = "ENABLE_HTTPS"
	flagEnableHTTPS    = "s"
	defaultEnableHTTPS = false

	envConfig     = "CONFIG"
	flagConfig    = "c"
	flagConfigAlt = "config"
)

// jsonConfigFile отражает структуру JSON-файла конфигурации.
type jsonConfigFile struct {
	ServerAddress   *string `json:"server_address"`
	BaseURL         *string `json:"base_url"`
	FileStoragePath *string `json:"file_storage_path"`
	DatabaseDSN     *string `json:"database_dsn"`
	EnableHTTPS     *bool   `json:"enable_https"`
	LogLevel        *string `json:"log_level"`
	SecretKey       *string `json:"secret_key"`
	AuditFile       *string `json:"audit_file"`
	AuditURL        *string `json:"audit_url"`
}

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
	EnableHTTPS     bool
}

// New создаёт Config, читая флаги из os.Args[1:].
// Возвращает ошибку, если указанный JSON-файл конфигурации не читается или невалиден.
func New() (*Config, error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	return NewWithFlagSet(fs, os.Args[1:])
}

// NewWithFlagSet создаёт Config из указанного FlagSet и аргументов.
// Используется в тестах для передачи произвольных флагов.
// Возвращает ошибку, если указанный JSON-файл конфигурации не читается или невалиден.
func NewWithFlagSet(fs *flag.FlagSet, args []string) (*Config, error) {
	// Объявляем флаги с настоящими значениями по умолчанию, чтобы -h
	// показывал корректную справку. Явность флага определяем через fs.Visit.
	serverAddr := fs.String(flagServerAddress, defaultServerAddress, "HTTP server address")
	baseURL := fs.String(flagBaseAddress, defaultBaseAddress, "base URL for shortened links")
	logLevel := fs.String(flagLogLevel, defaultLogLevel, "log level (Debug, Info, Warn, Error)")
	filePath := fs.String(flagFileStoragePath, defaultFileStoragePath, "path to file storage")
	databaseDSN := fs.String(flagDatabaseDSN, defaultDatabaseDSN, "database DSN")
	secretKey := fs.String(flagSecretKey, defaultSecretKey, "secret key for signing")
	auditFile := fs.String(flagAuditFile, defaultAuditFile, "audit log file path")
	auditURL := fs.String(flagAuditURL, defaultAuditURL, "audit log server URL")
	enableHTTPS := fs.Bool(flagEnableHTTPS, defaultEnableHTTPS, "enable HTTPS")
	var cfgPath string
	fs.StringVar(&cfgPath, flagConfig, "", "path to JSON config file")
	fs.StringVar(&cfgPath, flagConfigAlt, "", "path to JSON config file (alias for -c)")

	// Парсим флаги.
	_ = fs.Parse(args)

	// Собираем множество явно указанных флагов.
	explicitFlags := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		explicitFlags[f.Name] = true
	})

	// Конфигурационный файл — из флага или ENV.
	if !explicitFlags[flagConfig] && !explicitFlags[flagConfigAlt] {
		if envVal, ok := os.LookupEnv(envConfig); ok && envVal != "" {
			cfgPath = envVal
		}
	}

	// Загружаем JSON-файл, если указан.
	jsonCfg := &jsonConfigFile{}
	if cfgPath != "" {
		var err error
		jsonCfg, err = loadJSONConfig(cfgPath)
		if err != nil {
			return nil, fmt.Errorf("config file %s: %w", cfgPath, err)
		}
	}

	// Формируем итоговую конфигурацию.
	cfg := &Config{
		ServerAddress:   resolveString(envServerAddress, *serverAddr, explicitFlags[flagServerAddress], jsonCfg.ServerAddress, defaultServerAddress),
		BaseURL:         resolveString(envBaseAddress, *baseURL, explicitFlags[flagBaseAddress], jsonCfg.BaseURL, defaultBaseAddress),
		LogLevel:        resolveString(envLogLevel, *logLevel, explicitFlags[flagLogLevel], jsonCfg.LogLevel, defaultLogLevel),
		FileStoragePath: resolveString(envFileStoragePath, *filePath, explicitFlags[flagFileStoragePath], jsonCfg.FileStoragePath, defaultFileStoragePath),
		DatabaseDSN:     resolveString(envDatabaseDSN, *databaseDSN, explicitFlags[flagDatabaseDSN], jsonCfg.DatabaseDSN, defaultDatabaseDSN),
		SecretKey:       resolveString(envSecretKey, *secretKey, explicitFlags[flagSecretKey], jsonCfg.SecretKey, defaultSecretKey),
		AuditFile:       resolveString(envAuditFile, *auditFile, explicitFlags[flagAuditFile], jsonCfg.AuditFile, defaultAuditFile),
		AuditURL:        resolveString(envAuditURL, *auditURL, explicitFlags[flagAuditURL], jsonCfg.AuditURL, defaultAuditURL),
		EnableHTTPS:     resolveBool(envEnableHTTPS, *enableHTTPS, explicitFlags[flagEnableHTTPS], jsonCfg.EnableHTTPS, defaultEnableHTTPS),
	}

	return cfg, nil
}

// loadJSONConfig читает и парсит JSON-файл конфигурации.
// Возвращает ошибку, если файл не читается или содержит невалидный JSON.
func loadJSONConfig(path string) (*jsonConfigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	var cfg jsonConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	return &cfg, nil
}

// resolveString возвращает значение по цепочке приоритетов:
// ENV > явный флаг > JSON-файл > значение по умолчанию.
func resolveString(envName, flagValue string, flagSet bool, jsonValue *string, defaultValue string) string {
	if envVal, ok := os.LookupEnv(envName); ok {
		return envVal
	}
	if flagSet {
		return flagValue
	}
	if jsonValue != nil {
		return *jsonValue
	}
	return defaultValue
}

// resolveBool возвращает bool по цепочке приоритетов:
// ENV > явный флаг > JSON-файл > значение по умолчанию.
func resolveBool(envName string, flagValue bool, flagSet bool, jsonValue *bool, defaultValue bool) bool {
	if envVal, ok := os.LookupEnv(envName); ok {
		if b, err := strconv.ParseBool(envVal); err == nil {
			return b
		}
	}
	if flagSet {
		return flagValue
	}
	if jsonValue != nil {
		return *jsonValue
	}
	return defaultValue
}
