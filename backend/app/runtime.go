package app

import (
	"crane-system/config"
	"fmt"
	"os"
)

func EnsureRuntimeDirs(cfg *config.Config) error {
	for _, dir := range []string{cfg.Storage.UploadsDir, cfg.Backup.Dir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create runtime dir %s: %w", dir, err)
		}
	}

	return nil
}
