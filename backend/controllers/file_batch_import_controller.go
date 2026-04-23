package controllers

import (
	"archive/zip"
	"crane-system/database"
	"crane-system/models"
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
	Workstations  []batchUploadWorkstation `json:"workstations"`
	TotalPrograms int                      `json:"total_programs"`
	TotalFiles    int                      `json:"total_files"`
	TempDir       string                   `json:"temp_dir"`
}

type batchImportMapping struct {
	WorkstationName string `json:"workstation_name"`
	ProductionLineID *uint `json:"production_line_id"`
	VehicleModelID   *uint `json:"vehicle_model_id"`
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

var (
	batchTaskMu  sync.RWMutex
	batchTaskSeq int64 = 1
	batchTasks   = map[int64]*batchImportTaskStatus{}
)

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
	lineID, err := parseUintParam(c.PostForm("production_line_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "production_line_id参数格式错误"})
		return
	}
	if !authorizeLineAction(c, lineID, lineActionUpload) {
		return
	}

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
		if m.ProductionLineID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "production_line_id不能为空"})
			return
		}
		if !authorizeLineAction(c, *m.ProductionLineID, lineActionManage) {
			return
		}
		mappingByName[m.WorkstationName] = m
	}

	taskID := createBatchTask(preview.TotalPrograms)
	go runBatchImportTask(taskID, preview, mappingByName)

	c.JSON(http.StatusOK, gin.H{"task_id": taskID})
}

func runBatchImportTask(taskID int64, preview batchUploadPreview, mappingByName map[string]batchImportMapping) {
	for _, ws := range preview.Workstations {
		mapping, ok := mappingByName[ws.Name]
		if !ok || mapping.ProductionLineID == nil {
			for range ws.Programs {
				updateBatchTask(taskID, func(status *batchImportTaskStatus) {
					status.Processed++
					status.Failed++
					status.Progress = float64(status.Processed) * 100 / float64(max(status.Total, 1))
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
			code := fmt.Sprintf("BATCH-%d-%d", time.Now().Unix(), nextSeq)

			program := models.Program{
				Name:             prog.Name,
				Code:             code,
				ProductionLineID: *mapping.ProductionLineID,
				Status:           "in_progress",
			}
			if mapping.VehicleModelID != nil {
				program.VehicleModelID = *mapping.VehicleModelID
			}

			err := database.DB.Create(&program).Error
			updateBatchTask(taskID, func(status *batchImportTaskStatus) {
				if err != nil {
					status.Failed++
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
			status.ErrorMessage = "全部导入失败，请检查映射和数据"
		})
		return
	}

	updateBatchTask(taskID, func(status *batchImportTaskStatus) {
		status.Status = "completed"
		status.Progress = 100
	})
}

func GetTaskStatus(c *gin.Context) {
	taskID, err := strconv.ParseInt(c.Param("task_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务ID格式错误"})
		return
	}

	snapshot, ok := snapshotBatchTask(taskID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
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
