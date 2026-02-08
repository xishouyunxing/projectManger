package controllers

import (
	"archive/zip"
	"crane-system/database"
	"crane-system/models"
	"crane-system/utils"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const uploadDir = "./uploads"

func init() {
	// 创建上传目录
	os.MkdirAll(uploadDir, os.ModePerm)
}

func UploadFile(c *gin.Context) {
	// 支持多文件上传
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "解析表单失败"})
		return
	}
	
	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未找到文件"})
		return
	}

	programID, _ := strconv.Atoi(c.PostForm("program_id"))
	version := c.PostForm("version")
	description := c.PostForm("description")

	userID, _ := c.Get("user_id")
	var uploadedFiles []models.ProgramFile

	// 获取程序信息用于构建目录结构
	var program models.Program
	if err := database.DB.First(&program, programID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
		return
	}

	// 获取产线和车型信息
	var productionLine models.ProductionLine
	if err := database.DB.First(&productionLine, program.ProductionLineID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "生产线不存在"})
		return
	}

	var vehicleModel models.VehicleModel
	if err := database.DB.First(&vehicleModel, program.VehicleModelID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "车型不存在"})
		return
	}

	// 检查是否已有相同版本的文件
	var existingVersion models.ProgramVersion
	err = database.DB.Where("program_id = ? AND version = ?", programID, version).First(&existingVersion).Error
	
	isNewVersion := err != nil // 如果找不到相同版本，则为新版本

	for _, fileHeader := range files {
		// 生成新的目录结构路径
		programPath := utils.GenerateProgramPath(
			uploadDir,
			vehicleModel.Name,
			productionLine.Name,
			program.Code,
			program.Name,
			version,
		)
		
		// 确保目录存在
		if err := utils.EnsureDirectoryExists(programPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建目录失败"})
			return
		}

		// 生成完整文件路径
		filePath := filepath.Join(programPath, fileHeader.Filename)
		
		// 检查路径安全性
		if !utils.IsSafePath(uploadDir, filePath) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "文件路径不安全"})
			return
		}

		// 保存文件
		if err := c.SaveUploadedFile(fileHeader, filePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "文件保存失败"})
			return
		}

		// 生成相对路径用于存储
		relativePath, err := utils.GetRelativePath(uploadDir, filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "路径解析失败"})
			return
		}

		programFile := models.ProgramFile{
			ProgramID:   uint(programID),
			FileName:    fileHeader.Filename,
			FilePath:    relativePath,
			FileSize:    fileHeader.Size,
			FileType:    filepath.Ext(fileHeader.Filename),
			Version:     version,
			UploadedBy:  userID.(uint),
			Description: description,
		}

		if err := database.DB.Create(&programFile).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "文件记录创建失败"})
			return
		}

		uploadedFiles = append(uploadedFiles, programFile)

		// 如果是新版本或者版本内重新上传，创建版本记录
		if isNewVersion || len(files) > 1 {
			programVersion := models.ProgramVersion{
				ProgramID:  uint(programID),
				Version:    version,
				FileID:     programFile.ID,
				UploadedBy: userID.(uint),
				ChangeLog:  description,
				IsCurrent:  true,
			}

			// 将之前的版本设为非当前版本（仅对新版本）
			if isNewVersion {
				database.DB.Model(&models.ProgramVersion{}).
					Where("program_id = ?", programID).
					Update("is_current", false)
			}

			database.DB.Create(&programVersion)
		}
	}

	// 更新程序当前版本
	database.DB.Model(&models.Program{}).Where("id = ?", programID).Update("version", version)

	c.JSON(http.StatusOK, gin.H{
		"message": "文件上传成功",
		"files":   uploadedFiles,
		"isNewVersion": isNewVersion,
	})
}

func DownloadFile(c *gin.Context) {
	var file models.ProgramFile
	if err := database.DB.First(&file, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	filePath := filepath.Join(uploadDir, file.FilePath)
	
	// 检查文件是否存在
	if !utils.FileExists(filePath) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件已被移动或删除"})
		return
	}
	
	// 检查路径安全性
	if !utils.IsSafePath(uploadDir, filePath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件路径不安全"})
		return
	}
	
	c.FileAttachment(filePath, file.FileName)
}

func GetProgramFiles(c *gin.Context) {
	// 获取所有文件，并按版本分组
	var files []models.ProgramFile
	if err := database.DB.
		Preload("Uploader").
		Where("program_id = ?", c.Param("program_id")).
		Order("version DESC, created_at DESC").
		Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询文件失败"})
		return
	}

	// 获取版本信息
	var versions []models.ProgramVersion
	if err := database.DB.
		Preload("Uploader").
		Where("program_id = ?", c.Param("program_id")).
		Order("created_at DESC").
		Find(&versions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询版本失败"})
		return
	}

	// 按版本分组文件
	versionFiles := make(map[string][]models.ProgramFile)
	versionMap := make(map[string]*models.ProgramVersion)
	
	// 首先构建版本映射
	for _, version := range versions {
		versionMap[version.Version] = &version
		versionFiles[version.Version] = []models.ProgramFile{}
	}
	
	// 然后按版本分组文件
	for _, file := range files {
		if versionFiles[file.Version] == nil {
			versionFiles[file.Version] = []models.ProgramFile{}
		}
		versionFiles[file.Version] = append(versionFiles[file.Version], file)
	}

	// 构建响应数据
	var result []map[string]interface{}
	processedVersions := make(map[string]bool)
	
	for versionName, files := range versionFiles {
		// 避免重复处理
		if processedVersions[versionName] {
			continue
		}
		processedVersions[versionName] = true
		
		versionInfo := versionMap[versionName]
		
		var createdAt time.Time
		var uploader *models.User
		changeLog := ""
		isCurrent := false
		
		// 如果有版本信息，使用版本信息
		if versionInfo != nil {
			createdAt = versionInfo.CreatedAt
			uploader = &versionInfo.Uploader
			changeLog = versionInfo.ChangeLog
			isCurrent = versionInfo.IsCurrent
		} else if len(files) > 0 {
			// 如果版本没有明确信息，使用文件信息推断
			createdAt = files[0].CreatedAt
			uploader = &files[0].Uploader
			changeLog = files[0].Description
		}
		
		versionData := map[string]interface{}{
			"version":     versionName,
			"change_log":  changeLog,
			"is_current":  isCurrent,
			"created_at":  createdAt,
			"uploader":    uploader,
			"files":       files,
			"file_count":  len(files),
		}
		
		result = append(result, versionData)
	}

	// 按版本创建时间排序（最新版本在前）
	sort.Slice(result, func(i, j int) bool {
		timeI, okI := result[i]["created_at"].(time.Time)
		timeJ, okJ := result[j]["created_at"].(time.Time)
		if okI && okJ {
			return timeI.After(timeJ)
		}
		return false
	})

	// 如果没有明确的当前版本，将最新的版本设为当前版本
	if len(result) > 0 {
		hasCurrent := false
		for _, version := range result {
			if version["is_current"].(bool) {
				hasCurrent = true
				break
			}
		}
		// 如果没有任何版本被标记为当前版本，将最新的版本标记为当前版本
		if !hasCurrent {
			result[0]["is_current"] = true
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"versions": result,
		"total_versions": len(result),
	})
}

func DeleteFile(c *gin.Context) {
	var file models.ProgramFile
	if err := database.DB.First(&file, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 删除物理文件
	filePath := filepath.Join(uploadDir, file.FilePath)
	
	// 检查路径安全性
	if !utils.IsSafePath(uploadDir, filePath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件路径不安全"})
		return
	}
	
	// 删除文件（忽略错误）
	_ = utils.DeleteFile(filePath)

	// 删除数据库记录
	if err := database.DB.Delete(&file).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func GetProgramVersions(c *gin.Context) {
	var versions []models.ProgramVersion
	if err := database.DB.
		Preload("File").
		Preload("Uploader").
		Where("program_id = ?", c.Param("program_id")).
		Order("created_at DESC").
		Find(&versions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, versions)
}

func CreateVersion(c *gin.Context) {
	var version models.ProgramVersion
	if err := c.ShouldBindJSON(&version); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	version.UploadedBy = userID.(uint)

	if err := database.DB.Create(&version).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusCreated, version)
}

func ActivateVersion(c *gin.Context) {
	var version models.ProgramVersion
	if err := database.DB.First(&version, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "版本不存在"})
		return
	}

	// 将同程序的其他版本设为非当前版本
	database.DB.Model(&models.ProgramVersion{}).
		Where("program_id = ?", version.ProgramID).
		Update("is_current", false)

	// 激活当前版本
	version.IsCurrent = true
	database.DB.Save(&version)

	// 更新程序当前版本
	database.DB.Model(&models.Program{}).Where("id = ?", version.ProgramID).Update("version", version.Version)

	c.JSON(http.StatusOK, version)
}

// DownloadProgramLatestVersion 打包下载程序的最新版本文件
func DownloadProgramLatestVersion(c *gin.Context) {
	programID := c.Param("program_id")
	
	// 获取程序信息（用于文件名）
	var program models.Program
	if err := database.DB.First(&program, programID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
		return
	}

	// 获取程序的最新版本文件
	var files []models.ProgramFile
	if err := database.DB.
		Preload("Uploader").
		Where("program_id = ?", programID).
		Order("version DESC, created_at DESC").
		Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询文件失败"})
		return
	}

	if len(files) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "该程序暂无文件"})
		return
	}

	// 获取最新版本号
	latestVersion := files[0].Version
	
	// 筛选最新版本的所有文件
	var latestFiles []models.ProgramFile
	for _, file := range files {
		if file.Version == latestVersion {
			latestFiles = append(latestFiles, file)
		}
	}

	if len(latestFiles) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "最新版本暂无文件"})
		return
	}

	// 创建ZIP文件
	programCode := program.Code
	if programCode == "" {
		programCode = programID
	}
	zipFileName := fmt.Sprintf("%s_%s.zip", programCode, latestVersion)
	createAndDownloadZip(c, latestFiles, zipFileName)
}

// DownloadVersionFiles 批量下载某个版本的所有文件
func DownloadVersionFiles(c *gin.Context) {
	version := c.Param("version")
	programID := c.Query("program_id")
	
	if programID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少program_id参数"})
		return
	}

	// 获取程序信息（用于文件名）
	var program models.Program
	if err := database.DB.First(&program, programID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
		return
	}

	// 获取指定版本的所有文件
	var files []models.ProgramFile
	if err := database.DB.
		Preload("Uploader").
		Where("program_id = ? AND version = ?", programID, version).
		Order("created_at DESC").
		Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询文件失败"})
		return
	}

	if len(files) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "该版本暂无文件"})
		return
	}

	// 创建ZIP文件
	programCode := program.Code
	if programCode == "" {
		programCode = programID
	}
	zipFileName := fmt.Sprintf("%s_%s.zip", programCode, version)
	createAndDownloadZip(c, files, zipFileName)
}

// createAndDownloadZip 创建ZIP文件并发送给客户端下载
func createAndDownloadZip(c *gin.Context, files []models.ProgramFile, zipFileName string) {
	// 设置响应头
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipFileName))

	// 创建ZIP写入器
	zipWriter := zip.NewWriter(c.Writer)
	defer zipWriter.Close()

	for _, file := range files {
		// 构建完整文件路径
		filePath := filepath.Join(uploadDir, file.FilePath)
		
		// 检查文件是否存在
		if !utils.FileExists(filePath) {
			continue // 跳过不存在的文件
		}
		
		// 检查路径安全性
		if !utils.IsSafePath(uploadDir, filePath) {
			continue // 跳过不安全的路径
		}
		
		// 打开文件
		fileReader, err := os.Open(filePath)
		if err != nil {
			continue // 跳过无法打开的文件
		}
		fileReader.Close()

		// 在ZIP中创建文件
		// 使用原始文件名，避免路径冲突
		zipEntryName := file.FileName
		// 如果文件名可能有重复，可以加上时间戳前缀
		if len(files) > 1 {
			zipEntryName = fmt.Sprintf("%d_%s", file.ID, file.FileName)
		}

		zipFileWriter, err := zipWriter.Create(zipEntryName)
		if err != nil {
			continue // 跳过无法创建的文件
		}

		// 复制文件内容到ZIP
		_, err = fileReader.WriteTo(zipFileWriter)
		if err != nil {
			continue // 跳过复制失败的文件
		}
	}
}
