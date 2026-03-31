package config

import (
	"os"
	"testing"
)

func TestNormalizeDBDriver(t *testing.T) {
	cases := map[string]string{
		"":              "sqlite",
		"sqlite":        "sqlite",
		"POSTGRESQL":    "postgres",
		"mssql":         "sqlserver",
		"mariadb":       "mysql",
		"sqlserver":     "sqlserver",
		"invalid-thing": "",
	}
	for in, want := range cases {
		got := NormalizeDBDriver(in)
		if got != want {
			t.Errorf("NormalizeDBDriver(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLoadConfig(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-jwt-secret-at-least-32-chars-long")
	os.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")
	os.Setenv("DB_DRIVER", "sqlite")
	defer func() {
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("ENCRYPTION_KEY")
		os.Unsetenv("DB_DRIVER")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.ServerPort != "8080" {
		t.Errorf("Default ServerPort should be 8080, got %s", cfg.ServerPort)
	}
	if cfg.DBDriver != "sqlite" {
		t.Errorf("Expected sqlite driver, got %s", cfg.DBDriver)
	}
	if cfg.SQLitePath != "data/cqa.db" {
		t.Errorf("Default SQLitePath should be data/cqa.db, got %s", cfg.SQLitePath)
	}
	if cfg.MessageDataDir != "data/messages" {
		t.Errorf("Default MessageDataDir should be data/messages, got %s", cfg.MessageDataDir)
	}
}

func TestLoadConfigInvalidDBDriver(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-jwt-secret-at-least-32-chars-long")
	os.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")
	os.Setenv("DB_DRIVER", "oracle")
	defer func() {
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("ENCRYPTION_KEY")
		os.Unsetenv("DB_DRIVER")
	}()

	_, err := Load()
	if err == nil {
		t.Fatal("Load should fail for unsupported DB_DRIVER")
	}
}

func TestLoadConfigMySQLRequiresHost(t *testing.T) {
	os.Setenv("JWT_SECRET", "test-jwt-secret-at-least-32-chars-long")
	os.Setenv("ENCRYPTION_KEY", "12345678901234567890123456789012")
	os.Setenv("DB_DRIVER", "mysql")
	os.Setenv("DB_HOST", "")
	defer func() {
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("ENCRYPTION_KEY")
		os.Unsetenv("DB_DRIVER")
		os.Unsetenv("DB_HOST")
	}()

	_, err := Load()
	if err == nil {
		t.Fatal("Load should fail when DB_HOST empty for mysql")
	}
}

func TestLoadConfigMissingRequired(t *testing.T) {
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("ENCRYPTION_KEY")

	_, err := Load()
	if err == nil {
		t.Fatal("Load should fail with missing required vars")
	}
}

func TestIsProduction(t *testing.T) {
	cfg := &Config{Env: "production"}
	if !cfg.IsProduction() {
		t.Error("Should be production")
	}

	cfg.Env = "development"
	if cfg.IsProduction() {
		t.Error("Should not be production")
	}
}
