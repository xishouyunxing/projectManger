package migration

import (
	"archive/zip"
	"crane-system/database"
	"crane-system/models"
	"crane-system/utils"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// FileMigrationStatus 文件迁移状态
type FileMigrationStatus struct {
	TotalFiles    int     `json:"total_files"`
	MigratedFiles int     `json:"migrated_files"`
	FailedFiles   int     `json:"failed_files"`
	Progress      float64 `json:"progress"`
	CurrentFile   string  `json:"current_file"`
	Status        string  `json:"status"` // "running", "completed", "failed"
	StartTime     string  `json:"start_time"`
	EndTime       string  `json:"end_time,omitempty"`
	ErrorMsg      string  `json:"error_msg,omitempty"`
}

var migrationStatus *FileMigrationStatus

// GetMigrationStatus 获取迁移状态
func GetMigrationStatus() *FileMigrationStatus {
	if migrationStatus == nil {
		migrationStatus = &FileMigrationStatus{
			Status: "not_started",
		}
	}
	return migrationStatus
}

// MigrateFilesToNewStructure 迁移文件到新的目录结构
func MigrateFilesToNewStructure() error {
	// 检查是否已经在运行
	status := GetMigrationStatus()
	if status.Status == "running" {
		return fmt.Errorf("文件迁移正在进行中")
	}

	// 重置状态
	migrationStatus = &FileMigrationStatus{
		Status:    "running",
		StartTime: time.Now().Format(time.RFC3339),
	}

	// 确保备份目录存在
	backupDir := filepath.Join(utils.BackupDir, "file_migration", time.Now().Format("20060102_150405"))
	if err := utils.EnsureDirectoryExists(backupDir); err != nil {
		status.Status = "failed"
		status.ErrorMsg = fmt.Sprintf("创建备份目录失败: %v", err)
		return err
	}

	// 创建迁移前的完整备份
	log.Println("创建迁移前备份...")
	backupName := fmt.Sprintf("pre_migration_backup_%s.zip", time.Now().Format("20060102_150405"))
	backupPath := filepath.Join(utils.BackupDir, backupName)
	
	if utils.FileExists(utils.UploadDir) {
		if err := createZipBackup(utils.UploadDir, backupPath); err != nil {
			status.Status = "failed"
			status.ErrorMsg = fmt.Sprintf("创建备份失败: %v", err)
			return err
		}
		log.Printf("备份已创建: %s", backupPath)
	}

	// 查询所有需要迁移的文件
	var files []models.ProgramFile
	if err := database.DB.Find(&files).Error; err != nil {
		status.Status = "failed"
		status.ErrorMsg = fmt.Sprintf("查询文件记录失败: %v", err)
		return err
	}

	status.TotalFiles = len(files)
	if status.TotalFiles == 0 {
		status.Status = "completed"
		status.EndTime = time.Now().Format(time.RFC3339)
		log.Println("没有需要迁移的文件")
		return nil
	}

	log.Printf("开始迁移 %d 个文件...", status.TotalFiles)

	// 逐个迁移文件
	for i, file := range files {
		status.CurrentFile = file.FileName
		status.Progress = float64(i) / float64(status.TotalFiles) * 100

		// 检查文件是否已经是新路径格式（以车型/生产线开头的）
		if isAlreadyMigrated(file.FilePath) {
			log.Printf("文件 %s 已经是新格式，跳过", file.FileName)
			status.MigratedFiles++
			continue
		}

		// 迁移文件
		if err := migrateSingleFile(&file, backupDir); err != nil {
			log.Printf("迁移文件 %s 失败: %v", file.FileName, err)
			status.FailedFiles++
			status.ErrorMsg = fmt.Sprintf("迁移文件 %s 失败: %v", file.FileName, err)
		} else {
			status.MigratedFiles++
			log.Printf("文件 %s 迁移成功", file.FileName)
		}

		// 更新进度
		status.Progress = float64(i+1) / float64(status.TotalFiles) * 100
	}

	// 完成迁移
	status.Status = "completed"
	status.EndTime = time.Now().Format(time.RFC3339)
	
	log.Printf("文件迁移完成: 成功 %d, 失败 %d", status.MigratedFiles, status.FailedFiles)
	return nil
}

// migrateSingleFile 迁移单个文件
func migrateSingleFile(file *models.ProgramFile, backupDir string) error {
	// 获取文件的完整路径
	oldPath := filepath.Join(utils.UploadDir, file.FilePath)
	
	// 检查旧文件是否存在
	if !utils.FileExists(oldPath) {
		log.Printf("源文件不存在: %s", oldPath)
		return fmt.Errorf("源文件不存在")
	}

	// 获取程序信息
	var program models.Program
	if err := database.DB.First(&program, file.ProgramID).Error; err != nil {
		return fmt.Errorf("获取程序信息失败: %v", err)
	}

	// 获取产线信息
	var productionLine models.ProductionLine
	if err := database.DB.First(&productionLine, program.ProductionLineID).Error; err != nil {
		return fmt.Errorf("获取产线信息失败: %v", err)
	}

	// 获取车型信息（从程序中获取）
	var vehicleModel models.VehicleModel
	if err := database.DB.First(&vehicleModel, program.VehicleModelID).Error; err != nil {
		return fmt.Errorf("获取车型信息失败: %v", err)
	}

	// 生成新的目录路径
	newDir := utils.GenerateProgramPath(
		utils.UploadDir,
		vehicleModel.Name,
		productionLine.Name,
		program.Code,
		program.Name,
		file.Version,
	)

	// 确保新目录存在
	if err := utils.EnsureDirectoryExists(newDir); err != nil {
		return fmt.Errorf("创建目标目录失败: %v", err)
	}

	// 生成新的文件路径
	newPath := filepath.Join(newDir, file.FileName)

	// 备份原文件到备份目录
	backupFilePath := filepath.Join(backupDir, filepath.Base(file.FilePath))
	if err := utils.CopyFile(oldPath, backupFilePath); err != nil {
		return fmt.Errorf("备份原文件失败: %v", err)
	}

	// 移动文件到新位置
	if err := utils.MoveFile(oldPath, newPath); err != nil {
		return fmt.Errorf("移动文件失败: %v", err)
	}

	// 更新数据库中的文件路径
	relativePath, err := utils.GetRelativePath(utils.UploadDir, newPath)
	if err != nil {
		return fmt.Errorf("获取相对路径失败: %v", err)
	}

	if err := database.DB.Model(file).Update("file_path", relativePath).Error; err != nil {
		// 如果数据库更新失败，尝试恢复文件
		_ = utils.MoveFile(newPath, oldPath)
		return fmt.Errorf("更新数据库失败: %v", err)
	}

	return nil
}

// isAlreadyMigrated 检查文件是否已经是新的路径格式
func isAlreadyMigrated(filePath string) bool {
	// 新的路径格式应该包含车型/程序名称_程序编号/版本的层级结构
	// 这里简单检查是否包含下划线，因为新格式中程序编号和程序名称用下划线连接
	// 更准确的判断可以检查路径层级
	return filepath.Dir(filePath) != "." && filepath.Dir(filePath) != utils.UploadDir
}

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

// RollbackMigration 回滚迁移
func RollbackMigration() error {
	status := GetMigrationStatus()
	if status.Status != "completed" {
		return fmt.Errorf("只能回滚已完成的迁移")
	}

	// 查找最新的备份文件
	backupDir := filepath.Join(utils.BackupDir, "file_migration")
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("读取备份目录失败: %v", err)
	}

	var latestBackup string
	var latestTime time.Time

	for _, file := range files {
		if file.IsDir() {
			dirPath := filepath.Join(backupDir, file.Name())
			dirInfo, err := file.Info()
			if err != nil {
				continue
			}
			
			if dirInfo.ModTime().After(latestTime) {
				latestTime = dirInfo.ModTime()
				latestBackup = dirPath
			}
		}
	}

	if latestBackup == "" {
		return fmt.Errorf("未找到备份文件")
	}

	log.Printf("开始回滚迁移，备份目录: %s", latestBackup)

	// 恢复备份中的文件
	return restoreFromBackup(latestBackup)
}

// restoreFromBackup 从备份恢复文件
func restoreFromBackup(backupDir string) error {
	// 读取备份中的文件
	files, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("读取备份文件失败: %v", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		backupFilePath := filepath.Join(backupDir, file.Name())
		restoreFilePath := filepath.Join(utils.UploadDir, file.Name())

		// 恢复文件
		if err := utils.CopyFile(backupFilePath, restoreFilePath); err != nil {
			log.Printf("恢复文件 %s 失败: %v", file.Name(), err)
			continue
		}

		log.Printf("文件 %s 恢复成功", file.Name())
	}

	return nil
}