package controllers

import (
	"archive/zip"
	"crane-system/database"
	"crane-system/models"
	"crane-system/utils"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func GetProgramVersions(c *gin.Context) {
	targetProgramID, err := parseUintParam(c.Param("program_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??ID????"})
		return
	}
	targetProgram, targetProgramID, _, err := resolveProgramTarget(database.DB, targetProgramID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "?????"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
		return
	}

	c.JSON(http.StatusOK, versions)
}

func CreateVersion(c *gin.Context) {
	var req struct {
		ProgramID uint   `json:"program_id" binding:"required"`
		Version   string `json:"version" binding:"required"`
		FileID    uint   `json:"file_id" binding:"required"`
		ChangeLog string `json:"change_log"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Version = strings.TrimSpace(req.Version)
	if req.Version == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "version is required"})
		return
	}

	targetProgram, targetProgramID, _, err := resolveProgramTarget(database.DB, req.ProgramID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "?????"})
		return
	}
	if !authorizeLineAction(c, targetProgram.ProductionLineID, lineActionManage) {
		return
	}

	userID, _ := c.Get("user_id")
	uploadedBy := userID.(uint)

	var file models.ProgramFile
	if err := database.DB.Where("id = ? AND program_id = ?", req.FileID, targetProgramID).First(&file).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file_id does not belong to the target program"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
		return
	}
	if strings.TrimSpace(file.Version) != req.Version {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file version does not match requested version"})
		return
	}

	var version models.ProgramVersion
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		var existing models.ProgramVersion
		if err := tx.Where("program_id = ? AND version = ?", targetProgramID, req.Version).First(&existing).Error; err == nil {
			return fmt.Errorf("duplicate_version")
		} else if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}

		var versionCount int64
		if err := tx.Model(&models.ProgramVersion{}).Where("program_id = ?", targetProgramID).Count(&versionCount).Error; err != nil {
			return err
		}

		version = models.ProgramVersion{
			ProgramID:  targetProgramID,
			Version:    req.Version,
			FileID:     file.ID,
			UploadedBy: uploadedBy,
			ChangeLog:  req.ChangeLog,
			IsCurrent:  versionCount == 0,
		}
		if err := tx.Create(&version).Error; err != nil {
			return err
		}
		if version.IsCurrent {
			if err := tx.Model(&models.Program{}).Where("id = ?", targetProgramID).Update("version", version.Version).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		if err.Error() == "duplicate_version" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "version already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
		return
	}

	c.JSON(http.StatusCreated, version)
}

func UpdateVersion(c *gin.Context) {
	versionID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??ID????"})
		return
	}

	var version models.ProgramVersion
	if err := database.DB.First(&version, versionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "?????"})
		return
	}

	targetProgram, _, _, err := resolveProgramTarget(database.DB, version.ProgramID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "?????"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
		return
	}

	version.ChangeLog = payload.ChangeLog
	c.JSON(http.StatusOK, version)
}

func ActivateVersion(c *gin.Context) {
	versionID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??ID????"})
		return
	}

	var version models.ProgramVersion
	if err := database.DB.First(&version, versionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "?????"})
		return
	}

	targetProgram, _, _, err := resolveProgramTarget(database.DB, version.ProgramID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "?????"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "??????"})
		return
	}

	c.JSON(http.StatusOK, version)
}

func DownloadProgramLatestVersion(c *gin.Context) {
	targetProgramID, err := parseUintParam(c.Param("program_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??ID????"})
		return
	}
	targetProgram, targetProgramID, _, err := resolveProgramTarget(database.DB, targetProgramID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "?????"})
		return
	}
	if !authorizeLineAction(c, targetProgram.ProductionLineID, lineActionDownload) {
		return
	}
	program := targetProgram

	versionRecord, err := resolveDownloadVersion(database.DB, targetProgramID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "???????"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "??????"})
		return
	}

	var files []models.ProgramFile
	if err := database.DB.
		Preload("Uploader").
		Where("program_id = ? AND version = ?", targetProgramID, versionRecord.Version).
		Order("created_at DESC").
		Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "??????"})
		return
	}

	if len(files) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "????????"})
		return
	}

	programCode := program.Code
	if programCode == "" {
		programCode = strconv.FormatUint(uint64(targetProgramID), 10)
	}
	zipFileName := fmt.Sprintf("%s_%s.zip", programCode, versionRecord.Version)
	createAndDownloadZip(c, files, zipFileName)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "??ID????"})
		return
	}
	targetProgram, targetProgramID, _, err := resolveProgramTarget(database.DB, targetProgramID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "?????"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "??????"})
		return
	}

	if len(files) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "???????"})
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
	uploadDir := utils.UploadDir()
	type resolvedZipEntry struct {
		FilePath  string
		EntryName string
	}

	resolvedEntries := make([]resolvedZipEntry, 0, len(files))
	for _, file := range files {
		filePath := filepath.Join(uploadDir, file.FilePath)
		if !utils.FileExists(filePath) {
			c.JSON(http.StatusNotFound, gin.H{"error": "?????????"})
			return
		}
		if !utils.IsSafePath(uploadDir, filePath) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "???????"})
			return
		}

		zipEntryName := file.FileName
		if len(files) > 1 {
			zipEntryName = fmt.Sprintf("%d_%s", file.ID, file.FileName)
		}
		resolvedEntries = append(resolvedEntries, resolvedZipEntry{
			FilePath:  filePath,
			EntryName: zipEntryName,
		})
	}

	if len(resolvedEntries) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "??????????"})
		return
	}

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipFileName))
	c.Status(http.StatusOK)

	zipWriter := zip.NewWriter(c.Writer)
	for _, entry := range resolvedEntries {
		fileReader, err := os.Open(entry.FilePath)
		if err != nil {
			_ = c.Error(err)
			return
		}

		zipFileWriter, err := zipWriter.Create(entry.EntryName)
		if err != nil {
			_ = fileReader.Close()
			_ = c.Error(err)
			return
		}

		if _, err := io.Copy(zipFileWriter, fileReader); err != nil {
			_ = fileReader.Close()
			_ = c.Error(err)
			return
		}
		if err := fileReader.Close(); err != nil {
			_ = c.Error(err)
			return
		}
	}

	if err := zipWriter.Close(); err != nil {
		_ = c.Error(err)
	}
}
