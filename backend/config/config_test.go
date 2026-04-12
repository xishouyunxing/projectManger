package config

import (
	"path/filepath"
	"testing"
)

func TestLoadConfigPreservesCompatibleDefaults(t *testing.T) {
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_PORT", "")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_NAME", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("SERVER_PORT", "")
	t.Setenv("DEFAULT_PASSWORD", "")
	t.Setenv("FRONTEND_DIST", "")

	LoadConfig()

	if AppConfig.DBPort != "3307" {
		t.Fatalf("expected default DBPort 3307, got %s", AppConfig.DBPort)
	}

	if AppConfig.DBUser != "crane_user" {
		t.Fatalf("expected default DBUser crane_user, got %s", AppConfig.DBUser)
	}

	if AppConfig.JWTSecret == "" {
		t.Fatal("expected JWT secret fallback to be non-empty")
	}

	if AppConfig.DefaultPassword != "123456" {
		t.Fatalf("expected default password 123456, got %s", AppConfig.DefaultPassword)
	}

	if filepath.Clean(AppConfig.FrontendDist) != filepath.Clean("../frontend/dist") {
		t.Fatalf("expected frontend dist ../frontend/dist, got %s", AppConfig.FrontendDist)
	}

	if AppConfig.MaxUploadSize != 100*1024*1024 {
		t.Fatalf("expected max upload size 100MB, got %d", AppConfig.MaxUploadSize)
	}
}
