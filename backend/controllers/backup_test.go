package controllers

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"crane-system/config"
	"crane-system/middleware"
	"crane-system/utils"

	"github.com/gin-gonic/gin"
)

func TestIsPathWithinBackupDir(t *testing.T) {
	config.AppConfig = &config.Config{}
	config.AppConfig.Backup.Dir = filepath.Clean("./backups")
	backupRoot := filepath.Clean(utils.BackupDir())
	if err := os.MkdirAll(backupRoot, 0o755); err != nil {
		t.Fatalf("ensure backup root: %v", err)
	}

	insideFile := filepath.Join(backupRoot, "database_backup_20260423.sql")
	if !isPathWithinBackupDir(insideFile) {
		t.Fatalf("expected inside path to be allowed: %s", insideFile)
	}

	outsideFile := filepath.Join(backupRoot+"_evil", "database_backup_20260423.sql")
	if isPathWithinBackupDir(outsideFile) {
		t.Fatalf("expected sibling path to be rejected: %s", outsideFile)
	}

	traversalPath := filepath.Join(backupRoot, "..", "outside", "backup.sql")
	if isPathWithinBackupDir(traversalPath) {
		t.Fatalf("expected traversal path to be rejected: %s", traversalPath)
	}
}

func setupBackupRestoreFilesTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api")
	api.Use(func(c *gin.Context) {
		c.Set("user_id", uint(1))
		c.Set("user_role", "admin")
		c.Next()
	})
	api.Use(middleware.AdminMiddleware())
	api.POST("/backup/restore/files/:name", RestoreFiles)
	return r
}

func performBackupRestoreRequest(r *gin.Engine, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", path, nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func TestRestoreFilesKeepsExistingDataWhenRestorePackageInvalid(t *testing.T) {
	tmpRoot := t.TempDir()
	uploadsDir := filepath.Join(tmpRoot, "uploads")
	backupsDir := filepath.Join(tmpRoot, "backups")
	if err := os.MkdirAll(uploadsDir, 0o755); err != nil {
		t.Fatalf("mkdir uploads: %v", err)
	}
	if err := os.MkdirAll(backupsDir, 0o755); err != nil {
		t.Fatalf("mkdir backups: %v", err)
	}

	originalFile := filepath.Join(uploadsDir, "keep.txt")
	if err := os.WriteFile(originalFile, []byte("keep-me"), 0o644); err != nil {
		t.Fatalf("write original upload file: %v", err)
	}

	invalidBackupPath := filepath.Join(backupsDir, "files_backup_invalid.zip")
	if err := os.WriteFile(invalidBackupPath, []byte("not-a-zip"), 0o644); err != nil {
		t.Fatalf("write invalid backup zip: %v", err)
	}

	config.AppConfig = &config.Config{}
	config.AppConfig.Storage.UploadsDir = uploadsDir
	config.AppConfig.Backup.Dir = backupsDir

	if !utils.FileExists(filepath.Join(utils.UploadDir(), "keep.txt")) {
		t.Fatalf("expected original file to exist before restore")
	}

	r := setupBackupRestoreFilesTestRouter()
	resp := performBackupRestoreRequest(r, "/api/backup/restore/files/files_backup_invalid.zip")
	if resp.Code != 500 {
		t.Fatalf("expected 500, got %d body=%s", resp.Code, resp.Body.String())
	}

	if !utils.FileExists(filepath.Join(utils.UploadDir(), "keep.txt")) {
		t.Fatalf("expected original file to still exist after failed restore")
	}
}
