package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	BackupDir  = "./backups"
	UploadDir  = "./uploads"
)

// 文件存储结构相关工具函数

// SanitizeFilename 清理文件名，移除不安全字符
func SanitizeFilename(filename string) string {
	// 移除或替换不安全的字符
	reg := regexp.MustCompile(`[<>:"/\\|?*]`)
	sanitized := reg.ReplaceAllString(filename, "_")
	
	// 移除前后空格和点
	sanitized = strings.Trim(sanitized, " .")
	
	// 限制长度
	if len(sanitized) > 255 {
		sanitized = sanitized[:255]
	}
	
	// 确保不为空
	if sanitized == "" {
		sanitized = "unnamed"
	}
	
	return sanitized
}

// GenerateProgramPath 生成程序文件存储路径
// 格式：uploads/车型/生产线/程序编号_程序名称/版本/
func GenerateProgramPath(baseDir, vehicleModel, productionLine, programCode, programName, version string) string {
	return filepath.Join(
		baseDir,
		SanitizeFilename(vehicleModel),
		SanitizeFilename(productionLine),
		fmt.Sprintf("%s_%s", programCode, SanitizeFilename(programName)),
		SanitizeFilename(version),
	)
}

// GenerateFilePath 生成完整文件路径
func GenerateFilePath(baseDir, vehicleModel, productionLine, programCode, programName, version, filename string) string {
	programPath := GenerateProgramPath(baseDir, vehicleModel, productionLine, programCode, programName, version)
	return filepath.Join(programPath, filename)
}

// EnsureDirectoryExists 确保目录存在，如果不存在则创建
func EnsureDirectoryExists(dirPath string) error {
	return os.MkdirAll(dirPath, 0755)
}

// GetFileSize 获取文件大小
func GetFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// FileExists 检查文件是否存在
func FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// CopyFile 复制文件
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}

// MoveFile 移动文件
func MoveFile(src, dst string) error {
	// 确保目标目录存在
	dir := filepath.Dir(dst)
	if err := EnsureDirectoryExists(dir); err != nil {
		return err
	}
	
	return os.Rename(src, dst)
}

// DeleteFile 删除文件
func DeleteFile(filePath string) error {
	return os.Remove(filePath)
}

// DeleteDirectory 删除目录及其所有内容
func DeleteDirectory(dirPath string) error {
	return os.RemoveAll(dirPath)
}

// GetRelativePath 获取相对路径
func GetRelativePath(basePath, fullPath string) (string, error) {
	return filepath.Rel(basePath, fullPath)
}

// IsSafePath 检查路径是否安全（防止目录遍历攻击）
func IsSafePath(basePath, targetPath string) bool {
	relPath, err := filepath.Rel(basePath, targetPath)
	if err != nil {
		return false
	}
	
	// 检查是否包含 ".."
	if strings.Contains(relPath, "..") {
		return false
	}
	
	return true
}

// GetDirectorySize 获取目录大小
func GetDirectorySize(dirPath string) (int64, error) {
	var size int64
	
	err := filepath.Walk(dirPath, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	
	return size, err
}

// ListFiles 列出目录中的所有文件
func ListFiles(dirPath string) ([]string, error) {
	var files []string
	
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	
	return files, err
}

// CreateBackupInfo 创建备份信息
type BackupInfo struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
	Type      string `json:"type"` // "database" 或 "files"
}

// BackupStatus 备份状态
type BackupStatus struct {
	Status    string `json:"status"`    // "running", "completed", "failed"
	Progress  int    `json:"progress"`  // 0-100
	Message   string `json:"message"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time,omitempty"`
}