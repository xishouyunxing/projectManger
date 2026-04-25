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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type batchUploadProgramFile struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Path string `json:"path"`
}

type batchUploadProgram struct {
	Name  string                   `json:"name"`
	Files []batchUploadProgramFile `json:"files"`
}

type batchUploadWorkstation struct {
	Name     string               `json:"name"`
	Programs []batchUploadProgram `json:"programs"`
}

type batchUploadPreview struct {
	PreviewID     string                   `json:"preview_id"`
	Workstations  []batchUploadWorkstation `json:"workstations"`
	TotalPrograms int                      `json:"total_programs"`
	TotalFiles    int                      `json:"total_files"`
}

type batchUploadPreviewState struct {
	Preview    batchUploadPreview
	TempDir    string
	OwnerUser  uint
	ExpiresAt  time.Time
	CleanupJob *time.Timer
}

type batchImportMapping struct {
	WorkstationName  string `json:"workstation_name"`
	ProductionLineID *uint  `json:"production_line_id"`
	VehicleModelID   *uint  `json:"vehicle_model_id"`
}

type batchImportTaskStatus struct {
	Status       string    `json:"status"`
	Total        int       `json:"total"`
	Processed    int       `json:"processed"`
	Success      int       `json:"success"`
	Failed       int       `json:"failed"`
	Progress     float64   `json:"progress"`
	CurrentItem  string    `json:"current_item"`
	ErrorMessage string    `json:"error_message"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

const batchTaskStatusTTL = 30 * time.Minute
const batchImportInitialVersion = "v1"
const batchPreviewTTL = 30 * time.Minute

var (
	batchTaskMu     sync.RWMutex
	batchTaskSeq    int64 = 1
	batchTasks            = map[int64]*batchImportTaskStatus{}
	batchPreviewMu  sync.Mutex
	batchPreviewSeq int64 = 1
	batchPreviews         = map[string]*batchUploadPreviewState{}
)

func cleanupExpiredBatchPreviewsLocked(now time.Time) {
	for previewID, state := range batchPreviews {
		if state.ExpiresAt.After(now) {
			continue
		}
		if state.CleanupJob != nil {
			state.CleanupJob.Stop()
		}
		delete(batchPreviews, previewID)
		_ = os.RemoveAll(state.TempDir)
	}
}

func expireBatchPreview(previewID string) {
	batchPreviewMu.Lock()
	state, ok := batchPreviews[previewID]
	if ok {
		delete(batchPreviews, previewID)
	}
	batchPreviewMu.Unlock()

	if ok {
		_ = os.RemoveAll(state.TempDir)
	}
}

func createBatchPreview(preview batchUploadPreview, tempDir string, ownerUser uint) string {
	batchPreviewMu.Lock()
	defer batchPreviewMu.Unlock()

	now := time.Now()
	cleanupExpiredBatchPreviewsLocked(now)

	previewID := fmt.Sprintf("preview-%d-%d", now.UnixNano(), batchPreviewSeq)
	batchPreviewSeq++
	state := &batchUploadPreviewState{
		Preview:   preview,
		TempDir:   tempDir,
		OwnerUser: ownerUser,
		ExpiresAt: now.Add(batchPreviewTTL),
	}
	state.CleanupJob = time.AfterFunc(batchPreviewTTL, func() {
		expireBatchPreview(previewID)
	})
	batchPreviews[previewID] = state
	return previewID
}

func getBatchPreview(previewID string, ownerUser uint) (batchUploadPreview, string, error) {
	batchPreviewMu.Lock()
	defer batchPreviewMu.Unlock()

	cleanupExpiredBatchPreviewsLocked(time.Now())

	state, ok := batchPreviews[previewID]
	if !ok {
		return batchUploadPreview{}, "", fmt.Errorf("preview_not_found")
	}
	if state.OwnerUser != ownerUser {
		return batchUploadPreview{}, "", fmt.Errorf("preview_forbidden")
	}
	return state.Preview, state.TempDir, nil
}

func consumeBatchPreview(previewID string, ownerUser uint) (batchUploadPreview, string, error) {
	batchPreviewMu.Lock()
	defer batchPreviewMu.Unlock()

	cleanupExpiredBatchPreviewsLocked(time.Now())

	state, ok := batchPreviews[previewID]
	if !ok {
		return batchUploadPreview{}, "", fmt.Errorf("preview_not_found")
	}
	if state.OwnerUser != ownerUser {
		return batchUploadPreview{}, "", fmt.Errorf("preview_forbidden")
	}
	if state.CleanupJob != nil {
		state.CleanupJob.Stop()
	}
	delete(batchPreviews, previewID)
	return state.Preview, state.TempDir, nil
}

func cleanupExpiredBatchTasksLocked(now time.Time) {
	for taskID, status := range batchTasks {
		if status.ExpiresAt.IsZero() || status.ExpiresAt.After(now) {
			continue
		}
		delete(batchTasks, taskID)
	}
}

func createBatchTask(total int) int64 {
	batchTaskMu.Lock()
	defer batchTaskMu.Unlock()

	cleanupExpiredBatchTasksLocked(time.Now())
	taskID := batchTaskSeq
	batchTaskSeq++
	batchTasks[taskID] = &batchImportTaskStatus{
		Status:    "processing",
		Total:     total,
		ExpiresAt: time.Now().Add(batchTaskStatusTTL),
	}
	return taskID
}

func updateBatchTask(taskID int64, updater func(status *batchImportTaskStatus)) {
	batchTaskMu.Lock()
	defer batchTaskMu.Unlock()

	status, ok := batchTasks[taskID]
	if !ok {
		return
	}
	updater(status)
	status.ExpiresAt = time.Now().Add(batchTaskStatusTTL)
}

func snapshotBatchTask(taskID int64) (batchImportTaskStatus, bool) {
	batchTaskMu.Lock()
	defer batchTaskMu.Unlock()

	cleanupExpiredBatchTasksLocked(time.Now())
	status, ok := batchTasks[taskID]
	if !ok {
		return batchImportTaskStatus{}, false
	}
	return *status, true
}

func BatchUploadPrograms(c *gin.Context) {
	allowedLineIDs, statusCode, message := resolveAuthorizedLineIDs(c, lineActionManage)
	if statusCode != 0 {
		c.JSON(statusCode, gin.H{"error": message})
		return
	}
	if allowedLineIDs != nil && len(allowedLineIDs) == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "no manageable production lines"})
		return
	}

	userIDValue, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "?????"})
		return
	}
	userID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "??????"})
		return
	}

	if lineIDValue := strings.TrimSpace(c.PostForm("production_line_id")); lineIDValue != "" {
		lineID, err := parseUintParam(lineIDValue)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid production_line_id"})
			return
		}
		if !authorizeLineAction(c, lineID, lineActionManage) {
			return
		}
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "???zip??"})
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "????????"})
		return
	}
	defer src.Close()

	tempDir, err := os.MkdirTemp("", "program-batch-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????????"})
		return
	}
	cleanupTempDir := true
	defer func() {
		if cleanupTempDir {
			_ = os.RemoveAll(tempDir)
		}
	}()

	zipPath := filepath.Join(tempDir, "batch.zip")
	dst, err := os.Create(zipPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????????"})
		return
	}
	if _, err = io.Copy(dst, src); err != nil {
		_ = dst.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????????"})
		return
	}
	if err := dst.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????????"})
		return
	}

	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "zip??????"})
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

	preview := batchUploadPreview{TotalFiles: totalFiles}
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

	preview.PreviewID = createBatchPreview(preview, tempDir, userID)
	cleanupTempDir = false
	c.JSON(http.StatusOK, preview)
}

func BatchImportPrograms(c *gin.Context) {
	var payload struct {
		PreviewID string               `json:"preview_id"`
		Mappings  []batchImportMapping `json:"mappings"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(payload.PreviewID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "preview_id is required"})
		return
	}

	userIDValue, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "?????"})
		return
	}
	userID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "??????"})
		return
	}

	preview, tempDir, err := getBatchPreview(payload.PreviewID, userID)
	if err != nil {
		switch err.Error() {
		case "preview_not_found":
			c.JSON(http.StatusBadRequest, gin.H{"error": "preview expired or not found"})
		case "preview_forbidden":
			c.JSON(http.StatusForbidden, gin.H{"error": "preview does not belong to current user"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "????????"})
		}
		return
	}

	mappingByName := make(map[string]batchImportMapping, len(payload.Mappings))
	for _, mapping := range payload.Mappings {
		if mapping.ProductionLineID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "production_line_id is required"})
			return
		}
		if !authorizeLineAction(c, *mapping.ProductionLineID, lineActionManage) {
			return
		}
		mappingByName[mapping.WorkstationName] = mapping
	}
	for _, workstation := range preview.Workstations {
		mapping, ok := mappingByName[workstation.Name]
		if !ok || mapping.ProductionLineID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("?? %s ???????", workstation.Name)})
			return
		}
	}

	preview, tempDir, err = consumeBatchPreview(payload.PreviewID, userID)
	if err != nil {
		switch err.Error() {
		case "preview_not_found":
			c.JSON(http.StatusBadRequest, gin.H{"error": "preview expired or not found"})
		case "preview_forbidden":
			c.JSON(http.StatusForbidden, gin.H{"error": "preview does not belong to current user"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "????????"})
		}
		return
	}

	taskID := createBatchTask(preview.TotalPrograms)
	go runBatchImportTask(taskID, preview, tempDir, mappingByName, userID)

	c.JSON(http.StatusOK, gin.H{"task_id": taskID})
}

func runBatchImportTask(taskID int64, preview batchUploadPreview, tempDir string, mappingByName map[string]batchImportMapping, uploadedBy uint) {
	defer func() {
		if tempDir != "" {
			_ = os.RemoveAll(tempDir)
		}
	}()

	zipPath := filepath.Join(tempDir, "batch.zip")
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		updateBatchTask(taskID, func(status *batchImportTaskStatus) {
			status.Status = "failed"
			status.ErrorMessage = "???????????"
		})
		return
	}
	defer zr.Close()

	archiveFiles := make(map[string]*zip.File, len(zr.File))
	for _, file := range zr.File {
		archiveFiles[filepath.ToSlash(file.Name)] = file
	}

	for _, ws := range preview.Workstations {
		mapping, ok := mappingByName[ws.Name]
		if !ok || mapping.ProductionLineID == nil {
			for range ws.Programs {
				updateBatchTask(taskID, func(status *batchImportTaskStatus) {
					status.Processed++
					status.Failed++
					status.Progress = float64(status.Processed) * 100 / float64(max(status.Total, 1))
					if status.ErrorMessage == "" {
						status.ErrorMessage = fmt.Sprintf("?? %s ???????", ws.Name)
					}
				})
			}
			continue
		}

		for _, prog := range ws.Programs {
			nextSeq := 1
			updateBatchTask(taskID, func(status *batchImportTaskStatus) {
				status.CurrentItem = fmt.Sprintf("%s/%s", ws.Name, prog.Name)
				nextSeq = status.Processed + 1
			})

			err := importBatchProgramFiles(archiveFiles, prog, mapping, uploadedBy, nextSeq)
			updateBatchTask(taskID, func(status *batchImportTaskStatus) {
				if err != nil {
					status.Failed++
					if status.ErrorMessage == "" {
						status.ErrorMessage = err.Error()
					}
				} else {
					status.Success++
				}
				status.Processed++
				status.Progress = float64(status.Processed) * 100 / float64(max(status.Total, 1))
			})
		}
	}

	snapshot, ok := snapshotBatchTask(taskID)
	if !ok {
		return
	}
	if snapshot.Failed > 0 && snapshot.Success == 0 {
		updateBatchTask(taskID, func(status *batchImportTaskStatus) {
			status.Status = "failed"
			if status.ErrorMessage == "" {
				status.ErrorMessage = "???????????????"
			}
		})
		return
	}

	updateBatchTask(taskID, func(status *batchImportTaskStatus) {
		status.Status = "completed"
		status.Progress = 100
	})
}

func importBatchProgramFiles(archiveFiles map[string]*zip.File, prog batchUploadProgram, mapping batchImportMapping, uploadedBy uint, sequence int) error {
	if mapping.ProductionLineID == nil {
		return fmt.Errorf("???????")
	}
	if len(prog.Files) == 0 {
		return fmt.Errorf("?? %s ???????", prog.Name)
	}

	uploadDir := utils.UploadDir()
	if err := utils.EnsureDirectoryExists(uploadDir); err != nil {
		return err
	}

	var productionLine models.ProductionLine
	if err := database.DB.First(&productionLine, *mapping.ProductionLineID).Error; err != nil {
		return fmt.Errorf("????????")
	}

	vehicleModelName := ""
	vehicleModelID := uint(0)
	if mapping.VehicleModelID != nil {
		var vehicleModel models.VehicleModel
		if err := database.DB.First(&vehicleModel, *mapping.VehicleModelID).Error; err != nil {
			return fmt.Errorf("???????")
		}
		vehicleModelID = vehicleModel.ID
		vehicleModelName = vehicleModel.Name
	}

	version := batchImportInitialVersion
	code := fmt.Sprintf("BATCH-%d-%d", time.Now().UnixNano(), sequence)
	programPath := utils.GenerateProgramPath(uploadDir, vehicleModelName, productionLine.Name, code, prog.Name, version)
	if !utils.IsSafePath(uploadDir, programPath) {
		return fmt.Errorf("??????????")
	}
	if err := utils.EnsureDirectoryExists(programPath); err != nil {
		return err
	}

	createdFilePaths := make([]string, 0, len(prog.Files))
	defer func() {
		if len(createdFilePaths) == 0 {
			_ = os.RemoveAll(programPath)
		}
	}()

	program := models.Program{
		Name:             prog.Name,
		Code:             code,
		ProductionLineID: *mapping.ProductionLineID,
		VehicleModelID:   vehicleModelID,
		Version:          version,
		Status:           "in_progress",
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&program).Error; err != nil {
			return err
		}

		seenTargetPaths := map[string]struct{}{}
		var latestFile models.ProgramFile
		for _, importedFile := range prog.Files {
			archiveFile, ok := archiveFiles[filepath.ToSlash(importedFile.Path)]
			if !ok {
				return fmt.Errorf("???????? %s", importedFile.Path)
			}

			targetPath := filepath.Join(programPath, utils.SanitizeFilename(importedFile.Name))
			if !utils.IsSafePath(uploadDir, targetPath) {
				return fmt.Errorf("??????????")
			}
			if _, exists := seenTargetPaths[targetPath]; exists {
				return fmt.Errorf("?? %s ??????", prog.Name)
			}
			seenTargetPaths[targetPath] = struct{}{}

			if err := writeBatchImportFile(archiveFile, targetPath); err != nil {
				return err
			}
			createdFilePaths = append(createdFilePaths, targetPath)

			relativePath, err := utils.GetRelativePath(uploadDir, targetPath)
			if err != nil {
				return err
			}

			programFile := models.ProgramFile{
				ProgramID:   program.ID,
				FileName:    filepath.Base(targetPath),
				FilePath:    relativePath,
				FileSize:    importedFile.Size,
				FileType:    filepath.Ext(importedFile.Name),
				Version:     version,
				UploadedBy:  uploadedBy,
				Description: "??????",
			}
			if err := tx.Create(&programFile).Error; err != nil {
				return err
			}
			latestFile = programFile
		}

		versionRecord := models.ProgramVersion{
			ProgramID:  program.ID,
			Version:    version,
			FileID:     latestFile.ID,
			UploadedBy: uploadedBy,
			ChangeLog:  "?????????",
			IsCurrent:  true,
		}
		if err := tx.Create(&versionRecord).Error; err != nil {
			return err
		}

		if err := tx.Model(&models.Program{}).Where("id = ?", program.ID).Update("version", version).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		for _, filePath := range createdFilePaths {
			_ = os.Remove(filePath)
		}
		_ = os.RemoveAll(programPath)
		return err
	}

	createdFilePaths = nil
	return nil
}

func writeBatchImportFile(archiveFile *zip.File, targetPath string) error {
	reader, err := archiveFile.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	writer, err := os.Create(targetPath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(writer, reader); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
}

func GetTaskStatus(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("task_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??ID????"})
		return
	}

	snapshot, ok := snapshotBatchTask(taskID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "?????"})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
