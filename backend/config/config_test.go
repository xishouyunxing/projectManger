package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigUsesStructuredDefaults(t *testing.T) {
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_PORT", "")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_NAME", "")
	t.Setenv("JWT_SECRET", "12345678901234567890123456789012")
	t.Setenv("SERVER_PORT", "")
	t.Setenv("DEFAULT_PASSWORD", "admin123456")
	t.Setenv("FRONTEND_DIST", "")
	t.Setenv("UPLOADS_DIR", "")
	t.Setenv("BACKUPS_DIR", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("AUTO_MIGRATE", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	if err := LoadConfig(); err != nil {
		t.Fatalf("load config: %v", err)
	}

	if AppConfig.App.Env != "development" {
		t.Fatalf("expected app env development, got %s", AppConfig.App.Env)
	}

	if AppConfig.Database.Port != "3307" {
		t.Fatalf("expected default DB port 3307, got %s", AppConfig.Database.Port)
	}

	if AppConfig.Database.User != "crane_user" {
		t.Fatalf("expected default DB user crane_user, got %s", AppConfig.Database.User)
	}

	if AppConfig.Auth.JWTSecret != "12345678901234567890123456789012" {
		t.Fatalf("expected JWT secret from env, got %s", AppConfig.Auth.JWTSecret)
	}

	if AppConfig.Auth.DefaultPassword != "admin123456" {
		t.Fatalf("expected default password admin123456, got %s", AppConfig.Auth.DefaultPassword)
	}

	if filepath.Clean(AppConfig.App.FrontendDist) != filepath.Clean("../frontend/dist") {
		t.Fatalf("expected frontend dist ../frontend/dist, got %s", AppConfig.App.FrontendDist)
	}

	if filepath.Clean(AppConfig.Storage.UploadsDir) != filepath.Clean("uploads") {
		t.Fatalf("expected uploads dir ./uploads, got %s", AppConfig.Storage.UploadsDir)
	}

	if filepath.Clean(AppConfig.Backup.Dir) != filepath.Clean("backups") {
		t.Fatalf("expected backups dir ./backups, got %s", AppConfig.Backup.Dir)
	}

	if !AppConfig.App.AutoMigrate {
		t.Fatal("expected automigrate enabled outside production by default")
	}

	if AppConfig.Storage.MaxUploadSize != 100*1024*1024 {
		t.Fatalf("expected max upload size 100MB, got %d", AppConfig.Storage.MaxUploadSize)
	}

	if len(AppConfig.CORS.AllowedOrigins) != 2 {
		t.Fatalf("expected 2 default CORS origins, got %d", len(AppConfig.CORS.AllowedOrigins))
	}
}

func TestLoadConfigRejectsMissingSecrets(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("DEFAULT_PASSWORD", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")

	err := LoadConfig()
	if err == nil {
		t.Fatal("expected validation error")
	}

	if !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("expected JWT secret validation error, got %v", err)
	}
}

func TestLoadConfigRejectsWildcardCORS(t *testing.T) {
	t.Setenv("JWT_SECRET", "12345678901234567890123456789012")
	t.Setenv("DEFAULT_PASSWORD", "admin123456")
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")

	err := LoadConfig()
	if err == nil {
		t.Fatal("expected validation error")
	}

	if !strings.Contains(err.Error(), "通配符") {
		t.Fatalf("expected wildcard CORS validation error, got %v", err)
	}
}

func TestLoadConfigDisablesAutoMigrateByDefaultInProduction(t *testing.T) {
	t.Setenv("JWT_SECRET", "12345678901234567890123456789012")
	t.Setenv("DEFAULT_PASSWORD", "admin123456")
	t.Setenv("APP_ENV", "production")
	t.Setenv("AUTO_MIGRATE", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://example.com")

	if err := LoadConfig(); err != nil {
		t.Fatalf("load config: %v", err)
	}

	if AppConfig.App.AutoMigrate {
		t.Fatal("expected automigrate disabled by default in production")
	}
}
