package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"crane-system/utils"
	"errors"
	"fmt"
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

func buildStoredUploadFile(programPath, uploadDir, originalName string, reserved map[string]struct{}) (string, string, error) {
	displayName := utils.SanitizeFilename(filepath.Base(originalName))
	ext := filepath.Ext(displayName)
	baseName := strings.TrimSuffix(displayName, ext)
	if baseName == "" {
		baseName = "file"
	}

	for attempt := 0; ; attempt++ {
		storedName := displayName
		if attempt > 0 {
			storedName = fmt.Sprintf("%s__%d_%d%s", baseName, time.Now().UnixNano(), attempt, ext)
		}

		targetPath := filepath.Join(programPath, storedName)
		if !utils.IsSafePath(uploadDir, targetPath) {
			return "", "", errors.New("???????????")
		}
		if _, exists := reserved[targetPath]; exists {
			continue
		}
		if utils.FileExists(targetPath) {
			continue
		}

		reserved[targetPath] = struct{}{}
		return displayName, targetPath, nil
	}
}

func UploadFile(c *gin.Context) {
	uploadDir := utils.UploadDir()
	if err := utils.EnsureDirectoryExists(uploadDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建上传目录失败"})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize()+1024*1024)
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

	if err := validateMultipartUploadSize(files); err != nil {
		if errors.Is(err, errUploadTooLarge) {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "upload too large"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	vehicleModelName := ""
	if program.VehicleModelID > 0 {
		var vehicleModel models.VehicleModel
		if err := database.DB.First(&vehicleModel, program.VehicleModelID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "????????"})
			return
		}
		vehicleModelName = vehicleModel.Name
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
		reservedPaths := map[string]struct{}{}

		for _, fileHeader := range files {
			if fileHeader.Size <= 0 {
				return errors.New("文件不能为空")
			}
			programPath := utils.GenerateProgramPath(
				uploadDir,
				vehicleModelName,
				productionLine.Name,
				program.Code,
				program.Name,
				version,
			)

			if err := utils.EnsureDirectoryExists(programPath); err != nil {
				return err
			}

			displayName, filePath, err := buildStoredUploadFile(programPath, uploadDir, fileHeader.Filename, reservedPaths)
			if err != nil {
				return err
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
				FileName:    displayName,
				FilePath:    relativePath,
				FileSize:    fileHeader.Size,
				FileType:    filepath.Ext(displayName),
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

type programVersionHeader struct {
	ID        uint
	Version   string
	ChangeLog string
	IsCurrent bool
	CreatedAt time.Time
	Uploader  *models.User
}

func loadProgramVersionHeaders(programID uint) ([]programVersionHeader, error) {
	var versions []models.ProgramVersion
	if err := database.DB.
		Preload("Uploader").
		Where("program_id = ?", programID).
		Order("created_at DESC").
		Find(&versions).Error; err != nil {
		return nil, err
	}

	headers := make([]programVersionHeader, 0, len(versions))
	versionIndex := make(map[string]int, len(versions))
	for _, version := range versions {
		if idx, exists := versionIndex[version.Version]; exists {
			if version.CreatedAt.After(headers[idx].CreatedAt) {
				uploader := version.Uploader
				headers[idx] = programVersionHeader{
					ID:        version.ID,
					Version:   version.Version,
					ChangeLog: version.ChangeLog,
					IsCurrent: version.IsCurrent,
					CreatedAt: version.CreatedAt,
					Uploader:  &uploader,
				}
			}
			continue
		}

		uploader := version.Uploader
		headers = append(headers, programVersionHeader{
			ID:        version.ID,
			Version:   version.Version,
			ChangeLog: version.ChangeLog,
			IsCurrent: version.IsCurrent,
			CreatedAt: version.CreatedAt,
			Uploader:  &uploader,
		})
		versionIndex[version.Version] = len(headers) - 1
	}

	var fileOnlyVersionNames []string
	fileOnlyQuery := database.DB.Model(&models.ProgramFile{}).
		Distinct().
		Where("program_id = ?", programID)
	if len(versionIndex) > 0 {
		existingVersions := make([]string, 0, len(versionIndex))
		for version := range versionIndex {
			existingVersions = append(existingVersions, version)
		}
		fileOnlyQuery = fileOnlyQuery.Where("version NOT IN ?", existingVersions)
	}
	if err := fileOnlyQuery.
		Pluck("version", &fileOnlyVersionNames).Error; err != nil {
		return nil, err
	}

	for _, versionName := range fileOnlyVersionNames {
		if _, exists := versionIndex[versionName]; exists {
			continue
		}

		var latestFile models.ProgramFile
		if err := database.DB.Model(&models.ProgramFile{}).
			Select("version, created_at").
			Where("program_id = ? AND version = ?", programID, versionName).
			Order("created_at DESC").
			Take(&latestFile).Error; err != nil {
			return nil, err
		}

		headers = append(headers, programVersionHeader{
			Version:   versionName,
			CreatedAt: latestFile.CreatedAt,
		})
		versionIndex[versionName] = len(headers) - 1
	}

	sort.Slice(headers, func(i, j int) bool {
		return headers[i].CreatedAt.After(headers[j].CreatedAt)
	})

	if len(headers) > 0 {
		hasCurrent := false
		for _, header := range headers {
			if header.IsCurrent {
				hasCurrent = true
				break
			}
		}
		if !hasCurrent {
			headers[0].IsCurrent = true
		}
	}

	return headers, nil
}

func GetProgramFiles(c *gin.Context) {
	targetProgramID, err := parseUintParam(c.Param("program_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid program id"})
		return
	}
	targetProgram, targetProgramID, _, err := resolveProgramTarget(database.DB, targetProgramID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "program not found"})
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

	headers, err := loadProgramVersionHeaders(targetProgramID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query versions"})
		return
	}

	total := len(headers)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	pagedHeaders := headers[start:end]
	versionNames := make([]string, 0, len(pagedHeaders))
	for _, header := range pagedHeaders {
		versionNames = append(versionNames, header.Version)
	}

	versionFiles := make(map[string][]models.ProgramFile, len(pagedHeaders))
	if len(versionNames) > 0 {
		var files []models.ProgramFile
		if err := database.DB.
			Preload("Uploader").
			Where("program_id = ? AND version IN ?", targetProgramID, versionNames).
			Order("created_at DESC").
			Find(&files).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to query files"})
			return
		}

		for _, file := range files {
			versionFiles[file.Version] = append(versionFiles[file.Version], file)
		}
	}

	pagedVersions := make([]map[string]interface{}, 0, len(pagedHeaders))
	for _, header := range pagedHeaders {
		files := versionFiles[header.Version]
		createdAt := header.CreatedAt
		uploader := header.Uploader
		changeLog := header.ChangeLog

		if len(files) > 0 {
			if createdAt.IsZero() {
				createdAt = files[0].CreatedAt
			}
			if uploader == nil {
				fileUploader := files[0].Uploader
				uploader = &fileUploader
			}
			if changeLog == "" {
				changeLog = files[0].Description
			}
		}

		pagedVersions = append(pagedVersions, map[string]interface{}{
			"id":         header.ID,
			"version":    header.Version,
			"change_log": changeLog,
			"is_current": header.IsCurrent,
			"created_at": createdAt,
			"uploader":   uploader,
			"files":      files,
			"file_count": len(files),
		})
	}

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "??ID????"})
		return
	}

	var file models.ProgramFile
	if err := database.DB.First(&file, fileID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "?????"})
		return
	}

	var program models.Program
	if err := database.DB.First(&program, file.ProgramID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "?????"})
		return
	}
	if !authorizeLineAction(c, program.ProductionLineID, lineActionManage) {
		return
	}

	uploadDir := utils.UploadDir()
	filePath := filepath.Join(uploadDir, file.FilePath)
	if !utils.IsSafePath(uploadDir, filePath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "???????"})
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&models.ProgramFile{}, file.ID).Error; err != nil {
			return err
		}
		return reconcileProgramVersionState(tx, file.ProgramID)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
		return
	}

	_ = utils.DeleteFile(filePath)
	c.JSON(http.StatusOK, gin.H{"message": "????"})
}
