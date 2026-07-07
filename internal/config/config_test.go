package config

import (
	"flag"
	"testing"
)

func clearEnvForTest(t *testing.T) {
	t.Helper()
	t.Setenv(envServerAddress, "")
	t.Setenv(envBaseAddress, "")
	t.Setenv(envLogLevel, "")
	t.Setenv(envFileStoragePath, "")
	t.Setenv(envDatabaseDSN, "")
	t.Setenv(envSecretKey, "")
	t.Setenv(envAuditFile, "")
	t.Setenv(envAuditURL, "")
}

func TestNew_UsesDefaultValuesWhenNoEnvOrFlags(t *testing.T) {
	clearEnvForTest(t)

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg := NewWithFlagSet(fs, []string{})

	if cfg.ServerAddress != defaultServerAddress {
		t.Errorf("ServerAddress: expected %q, got %q", defaultServerAddress, cfg.ServerAddress)
	}
	if cfg.BaseURL != defaultBaseAddress {
		t.Errorf("BaseURL: expected %q, got %q", defaultBaseAddress, cfg.BaseURL)
	}
	if cfg.LogLevel != defaultLogLevel {
		t.Errorf("LogLevel: expected %q, got %q", defaultLogLevel, cfg.LogLevel)
	}

}

func TestNew_UsesEnvValues(t *testing.T) {
	clearEnvForTest(t)

	t.Setenv(envServerAddress, ":9090")
	t.Setenv(envBaseAddress, "http://yandex.ru")
	t.Setenv(envLogLevel, "Debug")
	t.Setenv(envSecretKey, "my-secret-key")

	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	cfg := NewWithFlagSet(fs, []string{})

	if cfg.ServerAddress != ":9090" {
		t.Errorf("ServerAddress: expected %q, got %q", ":9090", cfg.ServerAddress)
	}
	if cfg.BaseURL != "http://yandex.ru" {
		t.Errorf("BaseURL: expected %q, got %q", "http://yandex.ru", cfg.BaseURL)
	}
	if cfg.LogLevel != "Debug" {
		t.Errorf("LogLevel: expected %q, got %q", "Debug", cfg.LogLevel)
	}
	if cfg.SecretKey != "my-secret-key" {
		t.Errorf("SecretKey: expected %q, got %q", "my-secret-key", cfg.SecretKey)
	}
}

func TestNew_FlagOverridesDefault(t *testing.T) {
	clearEnvForTest(t)

	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	args := []string{"-a", ":3000", "-b", "http://yandex.ru", "-l", "Error"}
	cfg := NewWithFlagSet(fs, args)

	if cfg.ServerAddress != ":3000" {
		t.Errorf("ServerAddress: expected %q (from flag), got %q", ":3000", cfg.ServerAddress)
	}
	if cfg.BaseURL != "http://yandex.ru" {
		t.Errorf("BaseURL: expected %q (from flag), got %q", "http://yandex.ru", cfg.BaseURL)
	}
	if cfg.LogLevel != "Error" {
		t.Errorf("LogLevel: expected %q (from flag), got %q", "Error", cfg.LogLevel)
	}
}

func TestNew_EnvOverridesFlag(t *testing.T) {
	clearEnvForTest(t)

	// Устанавливаем ENV
	t.Setenv(envServerAddress, ":9090")
	t.Setenv(envBaseAddress, "http://env.com")

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	// Передаем флаги CLI, которые должны быть проигнорированы в пользу ENV
	args := []string{"-a", ":8081", "-b", "http://flag.com"}
	cfg := NewWithFlagSet(fs, args)

	if cfg.ServerAddress != ":9090" {
		t.Errorf("ServerAddress: expected %q (from env), got %q", ":9090", cfg.ServerAddress)
	}
	if cfg.BaseURL != "http://env.com" {
		t.Errorf("BaseURL: expected %q (from env), got %q", "http://env.com", cfg.BaseURL)
	}
}

func TestNew_EmptyEnvValueUsesDefault(t *testing.T) {
	clearEnvForTest(t)

	// Устанавливаем ENV в пустые строки
	t.Setenv(envServerAddress, "")
	t.Setenv(envBaseAddress, "")

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cfg := NewWithFlagSet(fs, []string{})

	if cfg.ServerAddress != defaultServerAddress {
		t.Errorf("ServerAddress: expected %q (default), got %q", defaultServerAddress, cfg.ServerAddress)
	}
	if cfg.BaseURL != defaultBaseAddress {
		t.Errorf("BaseURL: expected %q (default), got %q", defaultBaseAddress, cfg.BaseURL)
	}
}
