package controllers

import (
	"archive/zip"
	"context"
	"crane-system/config"
	"crane-system/utils"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	backupDir = "./backups"
)

func init() {
	// 创建备份目录
	os.MkdirAll(backupDir, os.ModePerm)
}

// CreateDatabaseBackup 创建数据库备份
func CreateDatabaseBackup(c *gin.Context) {
	// 生成备份文件名
	timestamp := time.Now().Format("20060102_150405")
	backupFileName := fmt.Sprintf("database_backup_%s.sql", timestamp)
	backupFilePath := filepath.Join(backupDir, backupFileName)

	// 确保备份目录存在
	if err := utils.EnsureDirectoryExists(backupDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建备份目录失败"})
		return
	}

	// 使用mysqldump备份数据库
	if err := createMySQLDump(backupFilePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库备份失败: " + err.Error()})
		return
	}

	// 获取文件大小
	fileSize, _ := utils.GetFileSize(backupFilePath)

	backupInfo := utils.BackupInfo{
		Name:      backupFileName,
		Path:      backupFilePath,
		Size:      fileSize,
		CreatedAt: time.Now().Format(time.RFC3339),
		Type:      "database",
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "数据库备份成功",
		"backup":  backupInfo,
	})
}

// CreateFilesBackup 创建文件系统备份
func CreateFilesBackup(c *gin.Context) {
	// 检查uploads目录是否存在
	if !utils.FileExists(utils.UploadDir) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件目录不存在"})
		return
	}

	// 生成备份文件名
	timestamp := time.Now().Format("20060102_150405")
	backupFileName := fmt.Sprintf("files_backup_%s.zip", timestamp)
	backupFilePath := filepath.Join(backupDir, backupFileName)

	// 获取目录大小
	dirSize, err := utils.GetDirectorySize(utils.UploadDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取文件大小失败"})
		return
	}

	// 创建ZIP备份文件
	if err := createZipBackup(utils.UploadDir, backupFilePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件备份失败: " + err.Error()})
		return
	}

	backupInfo := utils.BackupInfo{
		Name:      backupFileName,
		Path:      backupFilePath,
		Size:      dirSize,
		CreatedAt: time.Now().Format(time.RFC3339),
		Type:      "files",
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "文件备份成功",
		"backup":  backupInfo,
	})
}

// CreateFullBackup 创建完整系统备份
func CreateFullBackup(c *gin.Context) {
	timestamp := time.Now().Format("20060102_150405")
	backupFileName := fmt.Sprintf("full_backup_%s.zip", timestamp)
	backupFilePath := filepath.Join(backupDir, backupFileName)

	// 创建临时目录用于备份
	tempDir := filepath.Join(backupDir, "temp", timestamp)
	if err := utils.EnsureDirectoryExists(tempDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建临时目录失败"})
		return
	}
	defer os.RemoveAll(tempDir) // 清理临时目录

	// 备份数据库
	dbBackupName := fmt.Sprintf("database_%s.sql", timestamp)
	dbBackupPath := filepath.Join(tempDir, dbBackupName)
	if err := createMySQLDump(dbBackupPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库备份失败: " + err.Error()})
		return
	}

	// 备份文件目录
	filesBackupName := fmt.Sprintf("files_%s.zip", timestamp)
	filesBackupPath := filepath.Join(tempDir, filesBackupName)
	
	// 检查uploads目录是否存在，如果不存在或为空则创建空ZIP文件
	if utils.FileExists(utils.UploadDir) {
		if err := createZipBackup(utils.UploadDir, filesBackupPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "文件备份失败: " + err.Error()})
			return
		}
	} else {
		// 创建空的ZIP文件
		if err := createEmptyZip(filesBackupPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建空备份文件失败: " + err.Error()})
			return
		}
	}

	// 创建包含所有备份的ZIP文件
	if err := createFullSystemBackup(tempDir, backupFilePath, timestamp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "完整备份失败: " + err.Error()})
		return
	}

	// 获取备份文件大小
	fileSize, _ := utils.GetFileSize(backupFilePath)

	backupInfo := utils.BackupInfo{
		Name:      backupFileName,
		Path:      backupFilePath,
		Size:      fileSize,
		CreatedAt: time.Now().Format(time.RFC3339),
		Type:      "full",
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "完整系统备份成功",
		"backup":  backupInfo,
	})
}

// GetBackupList 获取备份文件列表
func GetBackupList(c *gin.Context) {
	var backups []utils.BackupInfo

	// 读取备份目录
	files, err := os.ReadDir(backupDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取备份目录失败"})
		return
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fileInfo, err := file.Info()
		if err != nil {
			continue
		}

		fileName := file.Name()
		filePath := filepath.Join(backupDir, fileName)
		
		// 确定备份类型
		backupType := "unknown"
		if strings.HasPrefix(fileName, "database_backup_") {
			backupType = "database"
		} else if strings.HasPrefix(fileName, "files_backup_") {
			backupType = "files"
		} else if strings.HasPrefix(fileName, "full_backup_") {
			backupType = "full"
		}

		backup := utils.BackupInfo{
			Name:      fileName,
			Path:      filePath,
			Size:      fileInfo.Size(),
			CreatedAt: fileInfo.ModTime().Format(time.RFC3339),
			Type:      backupType,
		}

		backups = append(backups, backup)
	}

	// 按创建时间降序排列（最新的在上面）
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt > backups[j].CreatedAt
	})

	c.JSON(http.StatusOK, gin.H{
		"backups": backups,
		"total":   len(backups),
	})
}

// DeleteBackup 删除备份文件
func DeleteBackup(c *gin.Context) {
	backupName := c.Param("name")
	backupPath := filepath.Join(backupDir, backupName)

	// 检查路径安全性
	if !strings.HasPrefix(filepath.Clean(backupPath), filepath.Clean(backupDir)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不安全的备份路径"})
		return
	}

	// 检查文件是否存在
	if !utils.FileExists(backupPath) {
		c.JSON(http.StatusNotFound, gin.H{"error": "备份文件不存在"})
		return
	}

	// 删除备份文件
	if err := utils.DeleteFile(backupPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除备份失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "备份删除成功"})
}

// RestoreDatabase 恢复数据库
func RestoreDatabase(c *gin.Context) {
	backupName := c.Param("name")
	backupPath := filepath.Join(backupDir, backupName)

	// 检查路径安全性
	if !strings.HasPrefix(filepath.Clean(backupPath), filepath.Clean(backupDir)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不安全的备份路径"})
		return
	}

	// 检查备份文件是否存在
	if !utils.FileExists(backupPath) {
		c.JSON(http.StatusNotFound, gin.H{"error": "备份文件不存在"})
		return
	}

	// 验证这是一个有效的数据库文件
	if !strings.HasPrefix(backupName, "database_backup_") && !strings.HasPrefix(backupName, "full_backup_") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的数据库备份文件"})
		return
	}

	// 创建当前数据库的备份作为回滚点
	currentBackupName := fmt.Sprintf("rollback_before_restore_%s.sql", time.Now().Format("20060102_150405"))
	currentBackupPath := filepath.Join(backupDir, currentBackupName)
	
	if err := createMySQLDump(currentBackupPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建回滚点失败: " + err.Error()})
		return
	}

	// 如果是完整备份，需要先解压
	if strings.HasPrefix(backupName, "full_backup_") {
		tempDir := filepath.Join(backupDir, "temp_restore", time.Now().Format("20060102_150405"))
		if err := utils.EnsureDirectoryExists(tempDir); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建临时目录失败"})
			return
		}
		defer os.RemoveAll(tempDir)

		// 解压完整备份
		if err := extractZipBackup(backupPath, tempDir); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "解压备份失败: " + err.Error()})
			return
		}

		// 查找数据库文件
		dbFiles, err := filepath.Glob(filepath.Join(tempDir, "database_*.sql"))
		if err != nil || len(dbFiles) == 0 {
			// 如果没找到database_*.sql，尝试查找database.sql
			dbFile := filepath.Join(tempDir, "database.sql")
			if _, err := os.Stat(dbFile); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "备份中未找到数据库文件"})
				return
			}
			dbFiles = []string{dbFile}
		}
		backupPath = dbFiles[0]
	}

	// 恢复数据库
	if err := restoreMySQLDump(backupPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库恢复失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "数据库恢复成功",
		"rollback_backup": currentBackupName,
	})
}

// RestoreFiles 恢复文件系统
func RestoreFiles(c *gin.Context) {
	backupName := c.Param("name")
	backupPath := filepath.Join(backupDir, backupName)

	// 检查路径安全性
	if !strings.HasPrefix(filepath.Clean(backupPath), filepath.Clean(backupDir)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不安全的备份路径"})
		return
	}

	// 检查备份文件是否存在
	if !utils.FileExists(backupPath) {
		c.JSON(http.StatusNotFound, gin.H{"error": "备份文件不存在"})
		return
	}

	// 验证这是一个有效的文件备份
	if !strings.HasPrefix(backupName, "files_backup_") && !strings.HasPrefix(backupName, "full_backup_") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件备份文件"})
		return
	}

	// 创建当前文件的备份作为回滚点
	currentBackupName := fmt.Sprintf("rollback_files_before_restore_%s.zip", time.Now().Format("20060102_150405"))
	currentBackupPath := filepath.Join(backupDir, currentBackupName)
	
	if utils.FileExists(utils.UploadDir) {
		if err := createZipBackup(utils.UploadDir, currentBackupPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建回滚点失败"})
			return
		}
	}

	// 如果是完整备份，需要先解压
	if strings.HasPrefix(backupName, "full_backup_") {
		tempDir := filepath.Join(backupDir, "temp_restore", time.Now().Format("20060102_150405"))
		if err := utils.EnsureDirectoryExists(tempDir); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建临时目录失败"})
			return
		}
		defer os.RemoveAll(tempDir)

		// 解压完整备份
		if err := extractZipBackup(backupPath, tempDir); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "解压备份失败: " + err.Error()})
			return
		}

		// 查找文件备份
		filesBackups, err := filepath.Glob(filepath.Join(tempDir, "files_*.zip"))
		if err != nil || len(filesBackups) == 0 {
			// 如果没找到files_*.zip，尝试查找files.zip
			filesBackup := filepath.Join(tempDir, "files.zip")
			if _, err := os.Stat(filesBackup); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "备份中未找到文件备份"})
				return
			}
			filesBackups = []string{filesBackup}
		}
		backupPath = filesBackups[0]
	}

	// 删除现有文件目录
	if utils.FileExists(utils.UploadDir) {
		if err := utils.DeleteDirectory(utils.UploadDir); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除现有文件失败"})
			return
		}
	}

	// 解压文件备份
	if err := extractZipBackup(backupPath, filepath.Dir(utils.UploadDir)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件恢复失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "文件系统恢复成功",
		"rollback_backup": currentBackupName,
	})
}

// DownloadBackup 下载备份文件
func DownloadBackup(c *gin.Context) {
	backupName := c.Param("name")
	backupPath := filepath.Join(backupDir, backupName)

	// 检查路径安全性
	if !strings.HasPrefix(filepath.Clean(backupPath), filepath.Clean(backupDir)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不安全的备份路径"})
		return
	}

	// 检查文件是否存在
	if !utils.FileExists(backupPath) {
		c.JSON(http.StatusNotFound, gin.H{"error": "备份文件不存在"})
		return
	}

	c.FileAttachment(backupPath, backupName)
}

// 辅助函数

// createZipBackup 创建ZIP备份
func createZipBackup(sourceDir, targetPath string) error {
	zipFile, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	return filepath.Walk(sourceDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 获取相对路径
		relPath, err := filepath.Rel(sourceDir, filePath)
		if err != nil {
			return err
		}

		// 如果是目录，创建目录条目
		if info.IsDir() {
			_, err = zipWriter.Create(relPath + "/")
			return err
		}

		// 创建文件条目
		fileWriter, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		// 打开源文件
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		// 复制文件内容
		_, err = io.Copy(fileWriter, file)
		return err
	})
}

// extractZipBackup 解压ZIP备份
func extractZipBackup(zipPath, targetDir string) error {
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	for _, file := range zipReader.File {
		filePath := filepath.Join(targetDir, file.Name)
		
		// 检查路径安全性
		if !strings.HasPrefix(filepath.Clean(filePath), filepath.Clean(targetDir)) {
			continue // 跳过不安全的路径
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, file.FileInfo().Mode())
			continue
		}

		// 创建父目录
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}

		// 打开ZIP中的文件
		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		// 创建目标文件
		targetFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.FileInfo().Mode())
		if err != nil {
			return err
		}
		defer targetFile.Close()

		// 复制文件内容
		_, err = io.Copy(targetFile, fileReader)
		if err != nil {
			return err
		}
	}

	return nil
}

// createFullSystemBackup 创建完整系统备份
func createFullSystemBackup(tempDir, targetPath, timestamp string) error {
	zipFile, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 添加数据库文件到ZIP
	dbFiles, err := filepath.Glob(filepath.Join(tempDir, "database_*.sql"))
	if err != nil {
		return fmt.Errorf("查找数据库文件失败: %v", err)
	}
	if len(dbFiles) == 0 {
		return fmt.Errorf("未找到数据库备份文件在目录: %s", tempDir)
	}
	if err := addFileToZip(zipWriter, dbFiles[0], "database.sql"); err != nil {
		return err
	}

	// 添加文件备份到ZIP
	filesBackups, err := filepath.Glob(filepath.Join(tempDir, "files_*.zip"))
	if err == nil && len(filesBackups) > 0 {
		if err := addFileToZip(zipWriter, filesBackups[0], "files.zip"); err != nil {
			return err
		}
	}

	// 添加备份信息文件
	infoContent := fmt.Sprintf(`Backup Type: Full System Backup
Created At: %s
System: %s
Go Version: %s
`, timestamp, runtime.GOOS, runtime.Version())

	infoWriter, err := zipWriter.Create("backup_info.txt")
	if err != nil {
		return err
	}
	_, err = infoWriter.Write([]byte(infoContent))
	return err
}

// addFileToZip 添加文件到ZIP
func addFileToZip(zipWriter *zip.Writer, filePath, entryName string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer, err := zipWriter.Create(entryName)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

// createMySQLDump 创建MySQL数据库备份
func createMySQLDump(backupPath string) error {
	// 构建mysqldump命令，移除需要额外权限的选项
	cmd := exec.CommandContext(context.Background(), "mysqldump",
		"-h", config.AppConfig.DBHost,
		"-P", config.AppConfig.DBPort,
		"-u", config.AppConfig.DBUser,
		fmt.Sprintf("-p%s", config.AppConfig.DBPassword),
		"--single-transaction",
		"--quick",
		"--lock-tables=false",
		config.AppConfig.DBName,
	)
	
	// 创建备份文件
	file, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("创建备份文件失败: %v", err)
	}
	defer file.Close()
	
	// 执行备份
	cmd.Stdout = file
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("执行mysqldump失败: %v", err)
	}
	
	return nil
}

// restoreMySQLDump 恢复MySQL数据库
func restoreMySQLDump(backupPath string) error {
	// 构建mysql命令
	cmd := exec.CommandContext(context.Background(), "mysql",
		"-h", config.AppConfig.DBHost,
		"-P", config.AppConfig.DBPort,
		"-u", config.AppConfig.DBUser,
		fmt.Sprintf("-p%s", config.AppConfig.DBPassword),
		config.AppConfig.DBName,
	)
	
	// 打开备份文件
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("打开备份文件失败: %v", err)
	}
	defer file.Close()
	
	// 执行恢复
	cmd.Stdin = file
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("执行mysql恢复失败: %v", err)
	}
	
	return nil
}

// createEmptyZip 创建空的ZIP文件
func createEmptyZip(targetPath string) error {
	zipFile, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 创建一个空的目录条目表示uploads目录
	_, err = zipWriter.Create("uploads/")
	return err
}