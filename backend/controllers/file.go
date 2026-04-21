package controllers

import (
	"archive/zip"
	"crane-system/database"
	"crane-system/models"
	"crane-system/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

	programID, _ := strconv.Atoi(c.PostForm("program_id"))
	version := c.PostForm("version")
	description := c.PostForm("description")

	userID, _ := c.Get("user_id")
	var uploadedFiles []models.ProgramFile

	targetProgram, targetProgramID, _, err := resolveProgramTarget(database.DB, uint(programID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
		return
	}
	program := targetProgram

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

	var latestUploadedFile models.ProgramFile
	var existingVersion models.ProgramVersion
	versionQueryErr := database.DB.Where("program_id = ? AND version = ?", targetProgramID, version).Order("created_at DESC").First(&existingVersion).Error
	isNewVersion := versionQueryErr != nil

	for _, fileHeader := range files {
		programPath := utils.GenerateProgramPath(
			uploadDir,
			vehicleModel.Name,
			productionLine.Name,
			program.Code,
			program.Name,
			version,
		)

		if err := utils.EnsureDirectoryExists(programPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建目录失败"})
			return
		}

		filePath := filepath.Join(programPath, fileHeader.Filename)
		if !utils.IsSafePath(uploadDir, filePath) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "文件路径不安全"})
			return
		}

		if err := c.SaveUploadedFile(fileHeader, filePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "文件保存失败"})
			return
		}

		relativePath, err := utils.GetRelativePath(uploadDir, filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "路径解析失败"})
			return
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

		if err := database.DB.Create(&programFile).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "文件记录创建失败"})
			return
		}

		uploadedFiles = append(uploadedFiles, programFile)
		latestUploadedFile = programFile
	}

	database.DB.Model(&models.ProgramVersion{}).
		Where("program_id = ?", targetProgramID).
		Update("is_current", false)

	if isNewVersion {
		if versionQueryErr != nil && versionQueryErr != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询版本信息失败"})
			return
		}

		programVersion := models.ProgramVersion{
			ProgramID:  targetProgramID,
			Version:    version,
			FileID:     latestUploadedFile.ID,
			UploadedBy: userID.(uint),
			ChangeLog:  description,
			IsCurrent:  true,
		}
		if err := database.DB.Create(&programVersion).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "版本记录创建失败"})
			return
		}
	} else {
		existingVersion.FileID = latestUploadedFile.ID
		existingVersion.UploadedBy = userID.(uint)
		existingVersion.IsCurrent = true
		if strings.TrimSpace(description) != "" {
			existingVersion.ChangeLog = description
		}
		if err := database.DB.Save(&existingVersion).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "版本记录更新失败"})
			return
		}
	}

	database.DB.Model(&models.Program{}).Where("id = ?", targetProgramID).Update("version", version)

	c.JSON(http.StatusOK, gin.H{
		"message":      "文件上传成功",
		"files":        uploadedFiles,
		"isNewVersion": isNewVersion,
	})
}

func DownloadFile(c *gin.Context) {
	var file models.ProgramFile
	if err := database.DB.First(&file, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
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
	targetProgram, targetProgramID, _, err := resolveProgramTarget(database.DB, parseUintParam(c.Param("program_id")))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
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

	var result []map[string]interface{}
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

		result = append(result, versionData)
	}

	sort.Slice(result, func(i, j int) bool {
		timeI, okI := result[i]["created_at"].(time.Time)
		timeJ, okJ := result[j]["created_at"].(time.Time)
		if okI && okJ {
			return timeI.After(timeJ)
		}
		return false
	})

	if len(result) > 0 {
		hasCurrent := false
		for _, version := range result {
			if version["is_current"].(bool) {
				hasCurrent = true
				break
			}
		}
		if !hasCurrent {
			result[0]["is_current"] = true
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"program_id":      targetProgram.ID,
		"versions":       result,
		"total_versions": len(result),
	})
}

func parseUintParam(value string) uint {
	parsed, _ := strconv.ParseUint(value, 10, 64)
	return uint(parsed)
}

func DeleteFile(c *gin.Context) {
	var file models.ProgramFile
	if err := database.DB.First(&file, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
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

func UpdateVersion(c *gin.Context) {
	var version models.ProgramVersion
	if err := database.DB.First(&version, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "版本不存在"})
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

type batchUploadProgramFile struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Path string `json:"path"`
}

type batchUploadProgram struct {
	Name  string                  `json:"name"`
	Files []batchUploadProgramFile `json:"files"`
}

type batchUploadWorkstation struct {
	Name     string              `json:"name"`
	Programs []batchUploadProgram `json:"programs"`
}

type batchUploadPreview struct {
	Workstations []batchUploadWorkstation `json:"workstations"`
	TotalPrograms int                     `json:"total_programs"`
	TotalFiles    int                     `json:"total_files"`
	TempDir       string                  `json:"temp_dir"`
}

type batchImportMapping struct {
	WorkstationName string `json:"workstation_name"`
	ProductionLineID *uint `json:"production_line_id"`
	VehicleModelID   *uint `json:"vehicle_model_id"`
}

type batchImportTaskStatus struct {
	Status      string  `json:"status"`
	Total       int     `json:"total"`
	Processed   int     `json:"processed"`
	Success     int     `json:"success"`
	Failed      int     `json:"failed"`
	Progress    float64 `json:"progress"`
	CurrentItem string  `json:"current_item"`
	ErrorMessage string `json:"error_message"`
}

var (
	batchTaskMu sync.RWMutex
	batchTaskSeq int64 = 1
	batchTasks = map[int64]*batchImportTaskStatus{}
)

func BatchUploadPrograms(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传zip文件"})
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取上传文件失败"})
		return
	}
	defer src.Close()

	tempDir, err := os.MkdirTemp("", "program-batch-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建临时目录失败"})
		return
	}

	zipPath := filepath.Join(tempDir, "batch.zip")
	dst, err := os.Create(zipPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建临时文件失败"})
		return
	}
	if _, err = io.Copy(dst, src); err != nil {
		dst.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存上传文件失败"})
		return
	}
	dst.Close()

	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "zip文件格式无效"})
		return
	}
	defer zr.Close()

	workstations := map[string]map[string][]batchUploadProgramFile{}
	totalFiles := 0

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		parts := strings.Split(strings.TrimPrefix(filepath.ToSlash(f.Name), "/"), "/")
		if len(parts) < 3 {
			continue
		}
		workstation := strings.TrimSpace(parts[0])
		program := strings.TrimSpace(parts[1])
		if workstation == "" || program == "" {
			continue
		}

		if _, ok := workstations[workstation]; !ok {
			workstations[workstation] = map[string][]batchUploadProgramFile{}
		}
		workstations[workstation][program] = append(workstations[workstation][program], batchUploadProgramFile{
			Name: filepath.Base(f.Name),
			Size: int64(f.UncompressedSize64),
			Path: filepath.ToSlash(f.Name),
		})
		totalFiles++
	}

	preview := batchUploadPreview{TempDir: tempDir, TotalFiles: totalFiles}
	for wsName, programs := range workstations {
		ws := batchUploadWorkstation{Name: wsName}
		for progName, files := range programs {
			ws.Programs = append(ws.Programs, batchUploadProgram{Name: progName, Files: files})
			preview.TotalPrograms++
		}
		sort.Slice(ws.Programs, func(i, j int) bool { return ws.Programs[i].Name < ws.Programs[j].Name })
		preview.Workstations = append(preview.Workstations, ws)
	}
	sort.Slice(preview.Workstations, func(i, j int) bool { return preview.Workstations[i].Name < preview.Workstations[j].Name })

	previewBytes, _ := json.Marshal(preview)
	_ = os.WriteFile(filepath.Join(tempDir, "preview.json"), previewBytes, 0644)

	c.JSON(http.StatusOK, preview)
}

func BatchImportPrograms(c *gin.Context) {
	var payload struct {
		TempDir  string               `json:"temp_dir"`
		Mappings []batchImportMapping `json:"mappings"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	previewPath := filepath.Join(payload.TempDir, "preview.json")
	previewRaw, err := os.ReadFile(previewPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "批量解析结果不存在，请重新上传"})
		return
	}

	var preview batchUploadPreview
	if err := json.Unmarshal(previewRaw, &preview); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "批量解析结果损坏"})
		return
	}

	mappingByName := make(map[string]batchImportMapping, len(payload.Mappings))
	for _, m := range payload.Mappings {
		mappingByName[m.WorkstationName] = m
	}

	batchTaskMu.Lock()
	taskID := batchTaskSeq
	batchTaskSeq++
	status := &batchImportTaskStatus{
		Status: "processing",
		Total:  preview.TotalPrograms,
	}
	batchTasks[taskID] = status
	batchTaskMu.Unlock()

	go runBatchImportTask(status, preview, mappingByName)

	c.JSON(http.StatusOK, gin.H{"task_id": taskID})
}

func runBatchImportTask(status *batchImportTaskStatus, preview batchUploadPreview, mappingByName map[string]batchImportMapping) {
	for _, ws := range preview.Workstations {
		mapping, ok := mappingByName[ws.Name]
		if !ok || mapping.ProductionLineID == nil {
			for range ws.Programs {
				status.Processed++
				status.Failed++
				status.Progress = float64(status.Processed) * 100 / float64(max(status.Total, 1))
			}
			continue
		}

		for _, prog := range ws.Programs {
			status.CurrentItem = fmt.Sprintf("%s/%s", ws.Name, prog.Name)
			code := fmt.Sprintf("BATCH-%d-%d", time.Now().Unix(), status.Processed+1)

			program := models.Program{
				Name:             prog.Name,
				Code:             code,
				ProductionLineID: *mapping.ProductionLineID,
				Status:           "in_progress",
			}
			if mapping.VehicleModelID != nil {
				program.VehicleModelID = *mapping.VehicleModelID
			}

			if err := database.DB.Create(&program).Error; err != nil {
				status.Failed++
			} else {
				status.Success++
			}
			status.Processed++
			status.Progress = float64(status.Processed) * 100 / float64(max(status.Total, 1))
		}
	}

	if status.Failed > 0 && status.Success == 0 {
		status.Status = "failed"
		status.ErrorMessage = "全部导入失败，请检查映射和数据"
		return
	}

	status.Status = "completed"
	status.Progress = 100
}

func GetTaskStatus(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("task_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务ID格式错误"})
		return
	}

	batchTaskMu.RLock()
	status, ok := batchTasks[taskID]
	batchTaskMu.RUnlock()
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	c.JSON(http.StatusOK, status)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func ActivateVersion(c *gin.Context) {
	var version models.ProgramVersion
	if err := database.DB.First(&version, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "版本不存在"})
		return
	}

	database.DB.Model(&models.ProgramVersion{}).
		Where("program_id = ?", version.ProgramID).
		Update("is_current", false)

	version.IsCurrent = true
	database.DB.Save(&version)

	database.DB.Model(&models.Program{}).Where("id = ?", version.ProgramID).Update("version", version.Version)

	c.JSON(http.StatusOK, version)
}

func DownloadProgramLatestVersion(c *gin.Context) {
	targetProgram, targetProgramID, _, err := resolveProgramTarget(database.DB, parseUintParam(c.Param("program_id")))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
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
	version := c.Param("version")
	programID := c.Query("program_id")

	if programID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少program_id参数"})
		return
	}

	targetProgram, targetProgramID, _, err := resolveProgramTarget(database.DB, parseUintParam(programID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
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
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipFileName))

	zipWriter := zip.NewWriter(c.Writer)
	defer zipWriter.Close()

	uploadDir := utils.UploadDir()
	for _, file := range files {
		filePath := filepath.Join(uploadDir, file.FilePath)
		if !utils.FileExists(filePath) {
			continue
		}
		if !utils.IsSafePath(uploadDir, filePath) {
			continue
		}

		fileReader, err := os.Open(filePath)
		if err != nil {
			continue
		}

		zipEntryName := file.FileName
		if len(files) > 1 {
			zipEntryName = fmt.Sprintf("%d_%s", file.ID, file.FileName)
		}

		zipFileWriter, err := zipWriter.Create(zipEntryName)
		if err != nil {
			fileReader.Close()
			continue
		}

		_, err = fileReader.WriteTo(zipFileWriter)
		fileReader.Close()
		if err != nil {
			continue
		}
	}
}
