package config

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// envVar описывает переменную окружения для теста.
type envVar struct {
	name  string
	value string
}

// configTestCase описывает один сценарий тестирования NewWithFlagSet.
type configTestCase struct {
	name      string
	args      []string
	envs      []envVar
	want      *Config
	wantErr   bool // ожидается ли ошибка от NewWithFlagSet
	wantHTTPS bool // опция для тестирования EnableHTTPS отдельно
}

func clearEnvForTest(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		envServerAddress, envBaseAddress, envLogLevel,
		envFileStoragePath, envDatabaseDSN, envSecretKey,
		envAuditFile, envAuditURL, envEnableHTTPS, envConfig,
	} {
		old, had := os.LookupEnv(name)
		os.Unsetenv(name)
		t.Cleanup(func() {
			if had {
				os.Setenv(name, old)
			} else {
				os.Unsetenv(name)
			}
		})
	}
}

func TestConfig(t *testing.T) {
	jsonPath := filepath.Join("testdata", "config.json")

	// Сохраняем дефаултные ожидаения
	def := &Config{
		ServerAddress:   defaultServerAddress,
		BaseURL:         defaultBaseAddress,
		LogLevel:        defaultLogLevel,
		FileStoragePath: defaultFileStoragePath,
		DatabaseDSN:     defaultDatabaseDSN,
		SecretKey:       defaultSecretKey,
		AuditFile:       defaultAuditFile,
		AuditURL:        defaultAuditURL,
		EnableHTTPS:     false,
	}

	// Ожидаемые значения из полного JSON
	fullJSON := &Config{
		ServerAddress:   "localhost:9090",
		BaseURL:         "http://json-config.com",
		FileStoragePath: "/path/to/file.db",
		LogLevel:        "Debug",
		SecretKey:       "json-secret",
		AuditFile:       "/tmp/audit.log",
		AuditURL:        "http://audit.example.com",
		EnableHTTPS:     true,
	}

	tests := []configTestCase{
		// === Дефаулты ===
		{
			name: "defaults without env/flags/json",
			args: []string{},
			want: def,
		},
		{
			name: "empty env values are respected as explicitly set",
			args: []string{},
			envs: []envVar{
				{envServerAddress, ""},
				{envBaseAddress, ""},
			},
			want: &Config{
				ServerAddress:   "",
				BaseURL:         "",
				LogLevel:        defaultLogLevel,
				FileStoragePath: defaultFileStoragePath,
				DatabaseDSN:     defaultDatabaseDSN,
				SecretKey:       defaultSecretKey,
				AuditFile:       defaultAuditFile,
				AuditURL:        defaultAuditURL,
				EnableHTTPS:     false,
			},
		},

		// === ENV ===
		{
			name: "env overrides defaults",
			args: []string{},
			envs: []envVar{
				{envServerAddress, ":9090"},
				{envBaseAddress, "http://yandex.ru"},
				{envLogLevel, "Debug"},
				{envSecretKey, "my-secret-key"},
			},
			want: mergeDefaults(def, &Config{
				ServerAddress: ":9090",
				BaseURL:       "http://yandex.ru",
				LogLevel:      "Debug",
				SecretKey:     "my-secret-key",
			}),
		},
		{
			name: "env overrides flag",
			args: []string{"-a", ":8081", "-b", "http://flag.com"},
			envs: []envVar{
				{envServerAddress, ":9090"},
				{envBaseAddress, "http://env.com"},
			},
			want: mergeDefaults(def, &Config{
				ServerAddress: ":9090",
				BaseURL:       "http://env.com",
			}),
		},
		{
			name: "env EnableHTTPS=true",
			args: []string{},
			envs: []envVar{{envEnableHTTPS, "true"}},
			want: mergeDefaults(def, &Config{EnableHTTPS: true}),
		},

		// === Флаги ===
		{
			name: "flags override defaults",
			args: []string{"-a", ":3000", "-b", "http://yandex.ru", "-l", "Error"},
			want: mergeDefaults(def, &Config{
				ServerAddress: ":3000",
				BaseURL:       "http://yandex.ru",
				LogLevel:      "Error",
			}),
		},
		{
			name: "flag -s enables HTTPS",
			args: []string{"-s"},
			want: mergeDefaults(def, &Config{EnableHTTPS: true}),
		},

		// === JSON ===
		{
			name: "full JSON via -c",
			args: []string{"-c", jsonPath},
			want: fullJSON,
		},
		{
			name: "full JSON via -config",
			args: []string{"-config", jsonPath},
			want: fullJSON,
		},
		{
			name: "full JSON via CONFIG env",
			args: []string{},
			envs: []envVar{{envConfig, jsonPath}},
			want: fullJSON,
		},
		{
			name:    "nonexistent JSON returns error",
			args:    []string{"-c", filepath.Join("testdata", "nonexistent.json")},
			wantErr: true,
		},

		// === Приоритет ===
		{
			name: "flag overrides JSON",
			args: []string{"-c", jsonPath, "-a", ":3000", "-b", "http://flag.com"},
			want: mergeDefaults(def, &Config{
				ServerAddress:   ":3000",
				BaseURL:         "http://flag.com",
				LogLevel:        "Debug",
				SecretKey:       "json-secret",
				AuditFile:       "/tmp/audit.log",
				AuditURL:        "http://audit.example.com",
				FileStoragePath: "/path/to/file.db",
				EnableHTTPS:     true,
			}),
		},
		{
			name: "env overrides JSON",
			args: []string{"-c", jsonPath},
			envs: []envVar{
				{envServerAddress, ":7070"},
				{envBaseAddress, "http://env.com"},
			},
			want: mergeDefaults(def, &Config{
				ServerAddress:   ":7070",
				BaseURL:         "http://env.com",
				LogLevel:        "Debug",
				SecretKey:       "json-secret",
				AuditFile:       "/tmp/audit.log",
				AuditURL:        "http://audit.example.com",
				FileStoragePath: "/path/to/file.db",
				EnableHTTPS:     true,
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnvForTest(t)

			// Устанавливаем переменные окружения
			for _, e := range tt.envs {
				t.Setenv(e.name, e.value)
			}

			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			cfg, err := NewWithFlagSet(fs, tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assertEqual(t, *tt.want, *cfg)
		})
	}
}

// mergeDefaults копирует def и перезаписывает не-нулевые поля из src.
func mergeDefaults(def, src *Config) *Config {
	out := *def
	if src.ServerAddress != "" {
		out.ServerAddress = src.ServerAddress
	}
	if src.BaseURL != "" {
		out.BaseURL = src.BaseURL
	}
	if src.LogLevel != "" {
		out.LogLevel = src.LogLevel
	}
	if src.FileStoragePath != "" {
		out.FileStoragePath = src.FileStoragePath
	}
	if src.DatabaseDSN != "" {
		out.DatabaseDSN = src.DatabaseDSN
	}
	if src.SecretKey != "" {
		out.SecretKey = src.SecretKey
	}
	if src.AuditFile != "" {
		out.AuditFile = src.AuditFile
	}
	if src.AuditURL != "" {
		out.AuditURL = src.AuditURL
	}
	if src.EnableHTTPS {
		out.EnableHTTPS = true
	}
	return &out
}

func assertEqual(t *testing.T, want, got Config) {
	t.Helper()
	if want.ServerAddress != got.ServerAddress {
		t.Errorf("ServerAddress: want %q, got %q", want.ServerAddress, got.ServerAddress)
	}
	if want.BaseURL != got.BaseURL {
		t.Errorf("BaseURL: want %q, got %q", want.BaseURL, got.BaseURL)
	}
	if want.LogLevel != got.LogLevel {
		t.Errorf("LogLevel: want %q, got %q", want.LogLevel, got.LogLevel)
	}
	if want.FileStoragePath != got.FileStoragePath {
		t.Errorf("FileStoragePath: want %q, got %q", want.FileStoragePath, got.FileStoragePath)
	}
	if want.DatabaseDSN != got.DatabaseDSN {
		t.Errorf("DatabaseDSN: want %q, got %q", want.DatabaseDSN, got.DatabaseDSN)
	}
	if want.SecretKey != got.SecretKey {
		t.Errorf("SecretKey: want %q, got %q", want.SecretKey, got.SecretKey)
	}
	if want.AuditFile != got.AuditFile {
		t.Errorf("AuditFile: want %q, got %q", want.AuditFile, got.AuditFile)
	}
	if want.AuditURL != got.AuditURL {
		t.Errorf("AuditURL: want %q, got %q", want.AuditURL, got.AuditURL)
	}
	if want.EnableHTTPS != got.EnableHTTPS {
		t.Errorf("EnableHTTPS: want %v, got %v", want.EnableHTTPS, got.EnableHTTPS)
	}
}

// TestNew_ConfigFromTempFile проверяет чтение JSON из временного файла.
func TestNew_ConfigFromTempFile(t *testing.T) {
	clearEnvForTest(t)

	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(`{"server_address": ":1234", "base_url": "http://tmp.com"}`)
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg, err := NewWithFlagSet(fs, []string{"-c", tmpFile.Name()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ServerAddress != ":1234" {
		t.Errorf("ServerAddress: want %q, got %q", ":1234", cfg.ServerAddress)
	}
	if cfg.BaseURL != "http://tmp.com" {
		t.Errorf("BaseURL: want %q, got %q", "http://tmp.com", cfg.BaseURL)
	}
}
