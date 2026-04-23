package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"crane-system/utils"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func UploadFile(c *gin.Context) {
	uploadDir := utils.UploadDir()
	if err := utils.EnsureDirectoryExists(uploadDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建上传目录失败"})
		return
	}

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

	programID, err := parseUintParam(c.PostForm("program_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "program_id参数格式错误"})
		return
	}
	version, err := parseRequiredString(c.PostForm("version"), "version")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	description := c.PostForm("description")

	userID, _ := c.Get("user_id")
	var uploadedFiles []models.ProgramFile
	uploadedFilePaths := make([]string, 0, len(files))
	isNewVersion := false

	targetProgram, targetProgramID, _, err := resolveProgramTarget(database.DB, programID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
		return
	}
	program := targetProgram
	if !authorizeLineAction(c, program.ProductionLineID, lineActionUpload) {
		return
	}

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

	uploadedFilePaths = uploadedFilePaths[:0]
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&models.Program{}, targetProgramID).Error; err != nil {
			return err
		}

		var latestUploadedFile models.ProgramFile
		var existingVersion models.ProgramVersion
		versionQueryErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("program_id = ? AND version = ?", targetProgramID, version).
			Order("created_at DESC").
			First(&existingVersion).Error
		isNewVersion = versionQueryErr != nil

		for _, fileHeader := range files {
			if fileHeader.Size <= 0 {
				return errors.New("文件不能为空")
			}
			programPath := utils.GenerateProgramPath(
				uploadDir,
				vehicleModel.Name,
				productionLine.Name,
				program.Code,
				program.Name,
				version,
			)

			if err := utils.EnsureDirectoryExists(programPath); err != nil {
				return err
			}

			filePath := filepath.Join(programPath, fileHeader.Filename)
			if !utils.IsSafePath(uploadDir, filePath) {
				return errors.New("文件路径不安全")
			}

			if err := c.SaveUploadedFile(fileHeader, filePath); err != nil {
				return err
			}
			uploadedFilePaths = append(uploadedFilePaths, filePath)

			relativePath, err := utils.GetRelativePath(uploadDir, filePath)
			if err != nil {
				return err
			}

			programFile := models.ProgramFile{
				ProgramID:   targetProgramID,
				FileName:    fileHeader.Filename,
				FilePath:    relativePath,
				FileSize:    fileHeader.Size,
				FileType:    filepath.Ext(fileHeader.Filename),
				Version:     version,
				UploadedBy:  userID.(uint),
				Description: description,
			}

			if err := tx.Create(&programFile).Error; err != nil {
				return err
			}

			uploadedFiles = append(uploadedFiles, programFile)
			latestUploadedFile = programFile
		}

		if err := tx.Model(&models.ProgramVersion{}).
			Where("program_id = ?", targetProgramID).
			Update("is_current", false).Error; err != nil {
			return err
		}

		if isNewVersion {
			if versionQueryErr != nil && versionQueryErr != gorm.ErrRecordNotFound {
				return versionQueryErr
			}

			programVersion := models.ProgramVersion{
				ProgramID:  targetProgramID,
				Version:    version,
				FileID:     latestUploadedFile.ID,
				UploadedBy: userID.(uint),
				ChangeLog:  description,
				IsCurrent:  true,
			}
			if err := tx.Create(&programVersion).Error; err != nil {
				return err
			}
		} else {
			existingVersion.FileID = latestUploadedFile.ID
			existingVersion.UploadedBy = userID.(uint)
			existingVersion.IsCurrent = true
			if strings.TrimSpace(description) != "" {
				existingVersion.ChangeLog = description
			}
			if err := tx.Save(&existingVersion).Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&models.Program{}).Where("id = ?", targetProgramID).Update("version", version).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		for _, filePath := range uploadedFilePaths {
			_ = os.Remove(filePath)
		}
		if err.Error() == "文件不能为空" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "文件不能为空"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件上传失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "文件上传成功",
		"files":        uploadedFiles,
		"isNewVersion": isNewVersion,
	})
}

func DownloadFile(c *gin.Context) {
	fileID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件ID格式错误"})
		return
	}

	var file models.ProgramFile
	if err := database.DB.First(&file, fileID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}
	var program models.Program
	if err := database.DB.First(&program, file.ProgramID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
		return
	}
	if !authorizeLineAction(c, program.ProductionLineID, lineActionDownload) {
		return
	}

	uploadDir := utils.UploadDir()
	filePath := filepath.Join(uploadDir, file.FilePath)
	if !utils.FileExists(filePath) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件已被移动或删除"})
		return
	}
	if !utils.IsSafePath(uploadDir, filePath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件路径不安全"})
		return
	}

	c.FileAttachment(filePath, file.FileName)
}

func GetProgramFiles(c *gin.Context) {
	targetProgramID, err := parseUintParam(c.Param("program_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "程序ID格式错误"})
		return
	}
	targetProgram, targetProgramID, _, err := resolveProgramTarget(database.DB, targetProgramID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
		return
	}
	if !authorizeLineAction(c, targetProgram.ProductionLineID, lineActionView) {
		return
	}

	page, err := parsePositiveIntQuery(c.Query("page"), 1, 0, "page")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pageSize, err := parsePositiveIntQuery(c.Query("page_size"), 20, 200, "page_size")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var files []models.ProgramFile
	if err := database.DB.
		Preload("Uploader").
		Where("program_id = ?", targetProgramID).
		Order("version DESC, created_at DESC").
		Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询文件失败"})
		return
	}

	var versions []models.ProgramVersion
	if err := database.DB.
		Preload("Uploader").
		Where("program_id = ?", targetProgramID).
		Order("created_at DESC").
		Find(&versions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询版本失败"})
		return
	}

	versionFiles := make(map[string][]models.ProgramFile)
	versionMap := make(map[string]models.ProgramVersion)

	for _, version := range versions {
		existing, ok := versionMap[version.Version]
		if !ok || version.CreatedAt.After(existing.CreatedAt) {
			versionMap[version.Version] = version
		}
		versionFiles[version.Version] = []models.ProgramFile{}
	}

	for _, file := range files {
		if versionFiles[file.Version] == nil {
			versionFiles[file.Version] = []models.ProgramFile{}
		}
		versionFiles[file.Version] = append(versionFiles[file.Version], file)
	}

	allVersions := make([]map[string]interface{}, 0, len(versionFiles))
	processedVersions := make(map[string]bool)

	for versionName, files := range versionFiles {
		if processedVersions[versionName] {
			continue
		}
		processedVersions[versionName] = true

		versionInfo, hasVersionInfo := versionMap[versionName]
		var createdAt time.Time
		var uploader *models.User
		changeLog := ""
		isCurrent := false

		if hasVersionInfo {
			createdAt = versionInfo.CreatedAt
			uploader = &versionInfo.Uploader
			changeLog = versionInfo.ChangeLog
			isCurrent = versionInfo.IsCurrent
		} else if len(files) > 0 {
			createdAt = files[0].CreatedAt
			uploader = &files[0].Uploader
			changeLog = files[0].Description
		}

		versionID := uint(0)
		if hasVersionInfo {
			versionID = versionInfo.ID
		}

		versionData := map[string]interface{}{
			"id":         versionID,
			"version":    versionName,
			"change_log": changeLog,
			"is_current": isCurrent,
			"created_at": createdAt,
			"uploader":   uploader,
			"files":      files,
			"file_count": len(files),
		}

		allVersions = append(allVersions, versionData)
	}

	sort.Slice(allVersions, func(i, j int) bool {
		timeI, okI := allVersions[i]["created_at"].(time.Time)
		timeJ, okJ := allVersions[j]["created_at"].(time.Time)
		if okI && okJ {
			return timeI.After(timeJ)
		}
		return false
	})

	if len(allVersions) > 0 {
		hasCurrent := false
		for _, version := range allVersions {
			if version["is_current"].(bool) {
				hasCurrent = true
				break
			}
		}
		if !hasCurrent {
			allVersions[0]["is_current"] = true
		}
	}

	total := len(allVersions)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	pagedVersions := allVersions[start:end]

	c.JSON(http.StatusOK, gin.H{
		"program_id":     targetProgram.ID,
		"versions":       pagedVersions,
		"total_versions": total,
		"page":           page,
		"page_size":      pageSize,
	})
}

func DeleteFile(c *gin.Context) {
	fileID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件ID格式错误"})
		return
	}

	var file models.ProgramFile
	if err := database.DB.First(&file, fileID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	var program models.Program
	if err := database.DB.First(&program, file.ProgramID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
		return
	}
	if !authorizeLineAction(c, program.ProductionLineID, lineActionManage) {
		return
	}

	uploadDir := utils.UploadDir()
	filePath := filepath.Join(uploadDir, file.FilePath)
	if !utils.IsSafePath(uploadDir, filePath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件路径不安全"})
		return
	}

	_ = utils.DeleteFile(filePath)

	if err := database.DB.Delete(&file).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
