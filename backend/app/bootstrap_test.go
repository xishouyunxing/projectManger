package app

import (
	"crane-system/config"
	"path/filepath"
	"testing"
)

func TestEnsureRuntimeDirsCreatesConfiguredDirectories(t *testing.T) {
	baseDir := t.TempDir()
	cfg := &config.Config{
		Storage: config.StorageSection{UploadsDir: filepath.Join(baseDir, "uploads")},
		Backup:  config.BackupSection{Dir: filepath.Join(baseDir, "backups")},
	}

	if err := EnsureRuntimeDirs(cfg); err != nil {
		t.Fatalf("ensure runtime dirs: %v", err)
	}

	for _, dir := range []string{cfg.Storage.UploadsDir, cfg.Backup.Dir} {
		if _, err := filepath.Abs(dir); err != nil {
			t.Fatalf("resolve dir %s: %v", dir, err)
		}
	}
}

func TestServerAddressUsesConfiguredPort(t *testing.T) {
	cfg := &config.Config{App: config.AppSection{ServerPort: "9090"}}

	if got := ServerAddress(cfg); got != ":9090" {
		t.Fatalf("expected :9090, got %s", got)
	}
}

func TestRunMigrationsIfEnabledSkipsWhenDisabled(t *testing.T) {
	cfg := &config.Config{App: config.AppSection{AutoMigrate: false}}

	if err := RunMigrationsIfEnabled(cfg); err != nil {
		t.Fatalf("expected nil when automigrate disabled, got %v", err)
	}
}
