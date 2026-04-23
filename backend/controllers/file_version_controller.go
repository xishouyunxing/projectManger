package controllers

import (
	"archive/zip"
	"bytes"
	"crane-system/database"
	"crane-system/models"
	"crane-system/utils"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func GetProgramVersions(c *gin.Context) {
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

	var versions []models.ProgramVersion
	if err := database.DB.
		Preload("File").
		Preload("Uploader").
		Where("program_id = ?", targetProgramID).
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

	targetProgram, targetProgramID, _, err := resolveProgramTarget(database.DB, version.ProgramID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
		return
	}
	if !authorizeLineAction(c, targetProgram.ProductionLineID, lineActionManage) {
		return
	}

	version.ProgramID = targetProgramID
	userID, _ := c.Get("user_id")
	version.UploadedBy = userID.(uint)

	if err := database.DB.Create(&version).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusCreated, version)
}

func UpdateVersion(c *gin.Context) {
	versionID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "版本ID格式错误"})
		return
	}

	var version models.ProgramVersion
	if err := database.DB.First(&version, versionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "版本不存在"})
		return
	}

	targetProgram, _, _, err := resolveProgramTarget(database.DB, version.ProgramID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
		return
	}
	if !authorizeLineAction(c, targetProgram.ProductionLineID, lineActionManage) {
		return
	}

	var payload struct {
		ChangeLog string `json:"change_log"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Model(&version).Update("change_log", payload.ChangeLog).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	version.ChangeLog = payload.ChangeLog
	c.JSON(http.StatusOK, version)
}

func ActivateVersion(c *gin.Context) {
	versionID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "版本ID格式错误"})
		return
	}

	var version models.ProgramVersion
	if err := database.DB.First(&version, versionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "版本不存在"})
		return
	}

	targetProgram, _, _, err := resolveProgramTarget(database.DB, version.ProgramID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
		return
	}
	if !authorizeLineAction(c, targetProgram.ProductionLineID, lineActionManage) {
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		var lockedVersion models.ProgramVersion
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lockedVersion, versionID).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.ProgramVersion{}).
			Where("program_id = ?", lockedVersion.ProgramID).
			Update("is_current", false).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.ProgramVersion{}).
			Where("id = ?", lockedVersion.ID).
			Update("is_current", true).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Program{}).
			Where("id = ?", lockedVersion.ProgramID).
			Update("version", lockedVersion.Version).Error; err != nil {
			return err
		}

		version = lockedVersion
		version.IsCurrent = true
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "激活版本失败"})
		return
	}

	c.JSON(http.StatusOK, version)
}

func DownloadProgramLatestVersion(c *gin.Context) {
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
	if !authorizeLineAction(c, targetProgram.ProductionLineID, lineActionDownload) {
		return
	}
	program := targetProgram

	var files []models.ProgramFile
	if err := database.DB.
		Preload("Uploader").
		Where("program_id = ?", targetProgramID).
		Order("version DESC, created_at DESC").
		Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询文件失败"})
		return
	}

	if len(files) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "该程序暂无文件"})
		return
	}

	latestVersion := files[0].Version
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

	programCode := program.Code
	if programCode == "" {
		programCode = strconv.FormatUint(uint64(targetProgramID), 10)
	}
	zipFileName := fmt.Sprintf("%s_%s.zip", programCode, latestVersion)
	createAndDownloadZip(c, latestFiles, zipFileName)
}

func DownloadVersionFiles(c *gin.Context) {
	version, err := parseRequiredString(c.Param("version"), "version")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	programID, err := parseRequiredString(c.Query("program_id"), "program_id")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	targetProgramID, err := parseUintParam(programID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "程序ID格式错误"})
		return
	}
	targetProgram, targetProgramID, _, err := resolveProgramTarget(database.DB, targetProgramID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
		return
	}
	if !authorizeLineAction(c, targetProgram.ProductionLineID, lineActionDownload) {
		return
	}
	program := targetProgram

	var files []models.ProgramFile
	if err := database.DB.
		Preload("Uploader").
		Where("program_id = ? AND version = ?", targetProgramID, version).
		Order("created_at DESC").
		Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询文件失败"})
		return
	}

	if len(files) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "该版本暂无文件"})
		return
	}

	programCode := program.Code
	if programCode == "" {
		programCode = strconv.FormatUint(uint64(targetProgramID), 10)
	}
	zipFileName := fmt.Sprintf("%s_%s.zip", programCode, version)
	createAndDownloadZip(c, files, zipFileName)
}

func createAndDownloadZip(c *gin.Context, files []models.ProgramFile, zipFileName string) {
	buffer := bytes.NewBuffer(nil)
	zipWriter := zip.NewWriter(buffer)

	uploadDir := utils.UploadDir()
	writtenEntries := 0
	for _, file := range files {
		filePath := filepath.Join(uploadDir, file.FilePath)
		if !utils.FileExists(filePath) {
			c.JSON(http.StatusNotFound, gin.H{"error": "文件已被移动或删除"})
			return
		}
		if !utils.IsSafePath(uploadDir, filePath) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "文件路径不安全"})
			return
		}

		fileReader, err := os.Open(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
			return
		}

		zipEntryName := file.FileName
		if len(files) > 1 {
			zipEntryName = fmt.Sprintf("%d_%s", file.ID, file.FileName)
		}

		zipFileWriter, err := zipWriter.Create(zipEntryName)
		if err != nil {
			_ = fileReader.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建压缩包失败"})
			return
		}

		if _, err := io.Copy(zipFileWriter, fileReader); err != nil {
			_ = fileReader.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "写入压缩包失败"})
			return
		}
		if err := fileReader.Close(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
			return
		}
		writtenEntries++
	}

	if writtenEntries == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "该版本暂无可下载文件"})
		return
	}
	if err := zipWriter.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成压缩包失败"})
		return
	}

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipFileName))
	c.Data(http.StatusOK, "application/zip", buffer.Bytes())
}
