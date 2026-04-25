package controllers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"crane-system/config"
	"crane-system/database"
	"crane-system/middleware"
	"crane-system/models"

	"github.com/gin-gonic/gin"
)

func setupProgramCustomFieldValueTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		programs := api.Group("/programs")
		{
			programs.GET("", GetPrograms)
			programs.GET("/export/excel", ExportProgramsExcel)
			programs.GET("/:id", GetProgram)
			programs.POST("", CreateProgram)
			programs.PUT("/:id", UpdateProgram)
			programs.PUT("/:id/custom-field-values", SaveProgramCustomFieldValues)
			programs.DELETE("/:id", DeleteProgram)
			programs.GET("/by-vehicle/:vehicle_id", GetProgramsByVehicle)
		}
		programRelations := api.Group("/program-relations")
		{
			programRelations.GET("/program/:program_id", GetProgramRelations)
			programRelations.DELETE("/:id", DeleteRelation)
		}
		mappings := api.Group("/program-mappings")
		{
			mappings.GET("/by-parent/:program_id", GetProgramMappingsByParent)
			mappings.GET("/by-child/:program_id", GetProgramMappingByChild)
			mappings.DELETE("/:id", DeleteProgramMapping)
		}
		batch := api.Group("/batch")
		{
			batch.POST("/import", BatchImportPrograms)
		}
		tasks := api.Group("/tasks")
		{
			tasks.GET("/:task_id/status", GetTaskStatus)
		}
		files := api.Group("/files")
		{
			files.GET("/:id/download", DownloadFile)
			files.GET("/download/version/:version", DownloadVersionFiles)
			files.GET("/program/:program_id", GetProgramFiles)
			files.DELETE("/:id", DeleteFile)
		}
		vehicleModels := api.Group("/vehicle-models")
		{
			vehicleModels.GET("", GetVehicleModels)
			vehicleModels.GET("/:id", GetVehicleModel)
		}
		versions := api.Group("/versions")
		{
			versions.POST("", CreateVersion)
			versions.PUT("/:id", UpdateVersion)
			versions.POST("/:id/activate", ActivateVersion)
		}
		users := api.Group("/users")
		{
			users.GET("/:id", GetUser)
			users.PUT("/:id", UpdateUser)
			users.DELETE("/:id", DeleteUser)
			users.POST("/:id/change-password", ChangePassword)
			users.POST("/:id/reset-password", ResetPassword)
		}
		permissions := api.Group("/permissions")
		{
			permissions.POST("", middleware.AdminMiddleware(), CreatePermission)
			permissions.GET("/user/:user_id", GetUserPermissions)
		}
		effectivePermissions := api.Group("/department-permissions")
		{
			effectivePermissions.POST("", middleware.AdminMiddleware(), CreateDepartmentPermission)
			effectivePermissions.GET("/user/:user_id/effective", GetUserEffectivePermissions)
		}
	}

	return r
}

func programCustomFieldValuesPath(programID uint) string {
	return fmt.Sprintf("/api/programs/%d/custom-field-values", programID)
}

func programDetailPath(programID uint) string {
	return fmt.Sprintf("/api/programs/%d", programID)
}

func programListPath() string {
	return "/api/programs"
}

func vehicleModelListPath(scope string) string {
	if strings.TrimSpace(scope) == "" {
		return "/api/vehicle-models"
	}
	return fmt.Sprintf("/api/vehicle-models?scope=%s", scope)
}

func vehicleModelDetailPath(vehicleModelID uint) string {
	return fmt.Sprintf("/api/vehicle-models/%d", vehicleModelID)
}

func programMappingsByParentPath(programID string) string {
	return fmt.Sprintf("/api/program-mappings/by-parent/%s", programID)
}

func programMappingsByChildPath(programID string) string {
	return fmt.Sprintf("/api/program-mappings/by-child/%s", programID)
}

func programMappingDetailPath(mappingID string) string {
	return fmt.Sprintf("/api/program-mappings/%s", mappingID)
}

func setupProgramCustomFieldValueTest(t *testing.T) (*gin.Engine, string, models.ProductionLine, models.Program) {
	t.Helper()
	database.DB = openProductionLineCustomFieldTestDB(t)
	token, line := seedProductionLineCustomFieldAuthData(t, database.DB)
	program := models.Program{Name: "程序A", Code: "PROG-001", ProductionLineID: line.ID, Status: "active"}
	if err := database.DB.Create(&program).Error; err != nil {
		t.Fatalf("create program: %v", err)
	}
	return setupProgramCustomFieldValueTestRouter(), token, line, program
}

func setupVehicleModelPermissionTest(t *testing.T) (*gin.Engine, string, models.ProductionLine, models.ProductionLine) {
	t.Helper()
	database.DB = openProductionLineCustomFieldTestDB(t)

	process := models.Process{Name: "总装", Code: "PROC-VM-001", Type: "upper"}
	if err := database.DB.Create(&process).Error; err != nil {
		t.Fatalf("create process: %v", err)
	}

	lineA := models.ProductionLine{Name: "产线A", Code: "LINE-VM-001", Type: "upper", Status: "active", ProcessID: &process.ID}
	if err := database.DB.Create(&lineA).Error; err != nil {
		t.Fatalf("create lineA: %v", err)
	}

	lineB := models.ProductionLine{Name: "产线B", Code: "LINE-VM-002", Type: "upper", Status: "active", ProcessID: &process.ID}
	if err := database.DB.Create(&lineB).Error; err != nil {
		t.Fatalf("create lineB: %v", err)
	}

	user := models.User{
		Name:       "Vehicle Viewer",
		Password:   "hashed",
		EmployeeID: "EMP-VM-001",
		Role:       "user",
		Status:     "active",
	}
	if err := database.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	permission := models.UserPermission{
		UserID:           user.ID,
		ProductionLineID: lineA.ID,
		CanView:          true,
	}
	if err := database.DB.Create(&permission).Error; err != nil {
		t.Fatalf("create permission: %v", err)
	}

	token := createUserTokenForTest(t, user.ID, "user")
	return setupProgramCustomFieldValueTestRouter(), token, lineA, lineB
}

func TestSaveProgramCustomFieldValuesReplacesCurrentSet(t *testing.T) {
	r, token, line, program := setupProgramCustomFieldValueTest(t)

	textField := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "备注", FieldType: "text", Enabled: true}
	if err := database.DB.Create(&textField).Error; err != nil {
		t.Fatalf("create text field: %v", err)
	}
	selectField := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "状态", FieldType: "select", OptionsJSON: `["试产","量产"]`, Enabled: true}
	if err := database.DB.Create(&selectField).Error; err != nil {
		t.Fatalf("create select field: %v", err)
	}
	clearedField := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "阶段", FieldType: "text", Enabled: true}
	if err := database.DB.Create(&clearedField).Error; err != nil {
		t.Fatalf("create cleared field: %v", err)
	}

	existingValues := []models.ProgramCustomFieldValue{
		{ProgramID: program.ID, ProductionLineCustomFieldID: textField.ID, Value: "旧备注"},
		{ProgramID: program.ID, ProductionLineCustomFieldID: clearedField.ID, Value: "待清空"},
	}
	for _, value := range existingValues {
		value := value
		if err := database.DB.Create(&value).Error; err != nil {
			t.Fatalf("create existing value: %v", err)
		}
	}

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, programCustomFieldValuesPath(program.ID), token, map[string]any{
		"values": []map[string]any{
			{"field_id": textField.ID, "value": "  新备注  "},
			{"field_id": selectField.ID, "value": "量产"},
		},
	})

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var saved []models.ProgramCustomFieldValue
	if err := database.DB.Where("program_id = ?", program.ID).Order("production_line_custom_field_id asc").Find(&saved).Error; err != nil {
		t.Fatalf("load saved values: %v", err)
	}

	if len(saved) != 2 {
		t.Fatalf("expected 2 saved values, got %d", len(saved))
	}
	if saved[0].ProductionLineCustomFieldID != textField.ID || saved[0].Value != "  新备注  " {
		t.Fatalf("unexpected first saved value: %#v", saved[0])
	}
	if saved[1].ProductionLineCustomFieldID != selectField.ID || saved[1].Value != "量产" {
		t.Fatalf("unexpected second saved value: %#v", saved[1])
	}

	var clearedCount int64
	if err := database.DB.Model(&models.ProgramCustomFieldValue{}).Where("program_id = ? AND production_line_custom_field_id = ?", program.ID, clearedField.ID).Count(&clearedCount).Error; err != nil {
		t.Fatalf("count cleared values: %v", err)
	}
	if clearedCount != 0 {
		t.Fatalf("expected omitted field value to be cleared, count=%d", clearedCount)
	}
}

func TestSaveProgramCustomFieldValuesValidatesInputTransactionally(t *testing.T) {
	t.Run("returns bad request when program id format is invalid", func(t *testing.T) {
		r, token, line, _ := setupProgramCustomFieldValueTest(t)
		field := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "备注", FieldType: "text", Enabled: true}
		if err := database.DB.Create(&field).Error; err != nil {
			t.Fatalf("create field: %v", err)
		}

		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, "/api/programs/abc/custom-field-values", token, map[string]any{
			"values": []map[string]any{{"field_id": field.ID, "value": "备注"}},
		})

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("returns not found when program does not exist", func(t *testing.T) {
		r, token, line, _ := setupProgramCustomFieldValueTest(t)
		field := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "备注", FieldType: "text", Enabled: true}
		if err := database.DB.Create(&field).Error; err != nil {
			t.Fatalf("create field: %v", err)
		}

		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, "/api/programs/999/custom-field-values", token, map[string]any{
			"values": []map[string]any{{"field_id": field.ID, "value": "备注"}},
		})

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("rejects missing values payload with bad request", func(t *testing.T) {
		r, token, _, program := setupProgramCustomFieldValueTest(t)

		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, programCustomFieldValuesPath(program.ID), token, map[string]any{})

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("rejects null values payload with bad request", func(t *testing.T) {
		r, token, _, program := setupProgramCustomFieldValueTest(t)

		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, programCustomFieldValuesPath(program.ID), token, map[string]any{"values": nil})

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("rejects fields from another production line without changing current values", func(t *testing.T) {
		r, token, line, program := setupProgramCustomFieldValueTest(t)
		validField := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "备注", FieldType: "text", Enabled: true}
		if err := database.DB.Create(&validField).Error; err != nil {
			t.Fatalf("create valid field: %v", err)
		}

		otherLine := models.ProductionLine{Name: "产线B", Code: "LINE-002", Type: "upper", Status: "active", ProcessID: line.ProcessID}
		if err := database.DB.Create(&otherLine).Error; err != nil {
			t.Fatalf("create other line: %v", err)
		}
		foreignField := models.ProductionLineCustomField{ProductionLineID: otherLine.ID, Name: "外部字段", FieldType: "text", Enabled: true}
		if err := database.DB.Create(&foreignField).Error; err != nil {
			t.Fatalf("create foreign field: %v", err)
		}

		existing := models.ProgramCustomFieldValue{ProgramID: program.ID, ProductionLineCustomFieldID: validField.ID, Value: "保留值"}
		if err := database.DB.Create(&existing).Error; err != nil {
			t.Fatalf("create existing value: %v", err)
		}

		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, programCustomFieldValuesPath(program.ID), token, map[string]any{
			"values": []map[string]any{{"field_id": foreignField.ID, "value": "非法"}},
		})

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
		}

		var saved []models.ProgramCustomFieldValue
		if err := database.DB.Where("program_id = ?", program.ID).Find(&saved).Error; err != nil {
			t.Fatalf("load saved values: %v", err)
		}
		if len(saved) != 1 || saved[0].ProductionLineCustomFieldID != validField.ID || saved[0].Value != "保留值" {
			t.Fatalf("expected existing values to remain untouched, got %#v", saved)
		}
	})

	t.Run("rejects zero field id with bad request", func(t *testing.T) {
		r, token, _, program := setupProgramCustomFieldValueTest(t)

		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, programCustomFieldValuesPath(program.ID), token, map[string]any{
			"values": []map[string]any{{"field_id": 0, "value": "非法"}},
		})

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("rejects duplicate field id with bad request", func(t *testing.T) {
		r, token, line, program := setupProgramCustomFieldValueTest(t)
		field := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "备注", FieldType: "text", Enabled: true}
		if err := database.DB.Create(&field).Error; err != nil {
			t.Fatalf("create field: %v", err)
		}

		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, programCustomFieldValuesPath(program.ID), token, map[string]any{
			"values": []map[string]any{
				{"field_id": field.ID, "value": "A"},
				{"field_id": field.ID, "value": "B"},
			},
		})

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("rejects invalid select option without changing current values", func(t *testing.T) {
		r, token, line, program := setupProgramCustomFieldValueTest(t)
		selectField := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "状态", FieldType: "select", OptionsJSON: `["试产","量产"]`, Enabled: true}
		if err := database.DB.Create(&selectField).Error; err != nil {
			t.Fatalf("create select field: %v", err)
		}

		existing := models.ProgramCustomFieldValue{ProgramID: program.ID, ProductionLineCustomFieldID: selectField.ID, Value: "试产"}
		if err := database.DB.Create(&existing).Error; err != nil {
			t.Fatalf("create existing value: %v", err)
		}

		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, programCustomFieldValuesPath(program.ID), token, map[string]any{
			"values": []map[string]any{{"field_id": selectField.ID, "value": "返工"}},
		})

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
		}

		var saved []models.ProgramCustomFieldValue
		if err := database.DB.Where("program_id = ?", program.ID).Find(&saved).Error; err != nil {
			t.Fatalf("load saved values: %v", err)
		}
		if len(saved) != 1 || saved[0].Value != "试产" {
			t.Fatalf("expected existing select value to remain untouched, got %#v", saved)
		}
	})

	t.Run("rejects disabled field writes", func(t *testing.T) {
		r, token, line, program := setupProgramCustomFieldValueTest(t)
		disabledField := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "停用字段", FieldType: "text", Enabled: true}
		if err := database.DB.Create(&disabledField).Error; err != nil {
			t.Fatalf("create disabled field: %v", err)
		}
		if err := database.DB.Model(&disabledField).Update("enabled", false).Error; err != nil {
			t.Fatalf("disable field: %v", err)
		}

		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, programCustomFieldValuesPath(program.ID), token, map[string]any{
			"values": []map[string]any{{"field_id": disabledField.ID, "value": "非法写入"}},
		})

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
		}

		var count int64
		if err := database.DB.Model(&models.ProgramCustomFieldValue{}).Where("program_id = ?", program.ID).Count(&count).Error; err != nil {
			t.Fatalf("count values: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected no values to be persisted, count=%d", count)
		}
	})
}

func TestDeleteProgramRemovesRelatedDataAndUploadedFiles(t *testing.T) {
	r, token, line, program := setupProgramCustomFieldValueTest(t)
	originalUploadDir := config.AppConfig.Storage.UploadsDir
	uploadDir := t.TempDir()
	config.AppConfig.Storage.UploadsDir = uploadDir
	t.Cleanup(func() { config.AppConfig.Storage.UploadsDir = originalUploadDir })

	field := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "??", FieldType: "text", Enabled: true}
	if err := database.DB.Create(&field).Error; err != nil {
		t.Fatalf("create field: %v", err)
	}
	value := models.ProgramCustomFieldValue{ProgramID: program.ID, ProductionLineCustomFieldID: field.ID, Value: "???"}
	if err := database.DB.Create(&value).Error; err != nil {
		t.Fatalf("create value: %v", err)
	}

	relatedProgram := models.Program{Name: "????", Code: "PROG-REL", ProductionLineID: line.ID, Status: "in_progress"}
	if err := database.DB.Create(&relatedProgram).Error; err != nil {
		t.Fatalf("create related program: %v", err)
	}
	filePath := filepath.Join("programs", "demo.nc")
	fullFilePath := filepath.Join(uploadDir, filePath)
	if err := os.MkdirAll(filepath.Dir(fullFilePath), 0755); err != nil {
		t.Fatalf("create upload dir: %v", err)
	}
	if err := os.WriteFile(fullFilePath, []byte("demo"), 0644); err != nil {
		t.Fatalf("write upload file: %v", err)
	}
	file := models.ProgramFile{ProgramID: program.ID, FileName: "demo.nc", FilePath: filePath, FileSize: 4, Version: "v1", UploadedBy: 1}
	if err := database.DB.Create(&file).Error; err != nil {
		t.Fatalf("create program file: %v", err)
	}
	version := models.ProgramVersion{ProgramID: program.ID, Version: "v1", FileID: file.ID, UploadedBy: 1, IsCurrent: true}
	if err := database.DB.Create(&version).Error; err != nil {
		t.Fatalf("create program version: %v", err)
	}
	mapping := models.ProgramMapping{ParentProgramID: program.ID, ChildProgramID: relatedProgram.ID, CreatedBy: 1}
	if err := database.DB.Create(&mapping).Error; err != nil {
		t.Fatalf("create mapping: %v", err)
	}
	relation := models.ProgramRelation{SourceProgramID: program.ID, RelatedProgramID: relatedProgram.ID, RelationType: "same_program"}
	if err := database.DB.Create(&relation).Error; err != nil {
		t.Fatalf("create relation: %v", err)
	}

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodDelete, programDetailPath(program.ID), token, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	countDeletedProgramRows := func(model any, where string, args ...any) int64 {
		var count int64
		if err := database.DB.Model(model).Where(where, args...).Count(&count).Error; err != nil {
			t.Fatalf("count related rows: %v", err)
		}
		return count
	}
	if count := countDeletedProgramRows(&models.ProgramCustomFieldValue{}, "program_id = ?", program.ID); count != 0 {
		t.Fatalf("expected custom field values deleted, count=%d", count)
	}
	if count := countDeletedProgramRows(&models.ProgramFile{}, "program_id = ?", program.ID); count != 0 {
		t.Fatalf("expected files deleted, count=%d", count)
	}
	if count := countDeletedProgramRows(&models.ProgramVersion{}, "program_id = ?", program.ID); count != 0 {
		t.Fatalf("expected versions deleted, count=%d", count)
	}
	if count := countDeletedProgramRows(&models.ProgramMapping{}, "parent_program_id = ? OR child_program_id = ?", program.ID, program.ID); count != 0 {
		t.Fatalf("expected mappings deleted, count=%d", count)
	}
	if count := countDeletedProgramRows(&models.ProgramRelation{}, "source_program_id = ? OR related_program_id = ?", program.ID, program.ID); count != 0 {
		t.Fatalf("expected relations deleted, count=%d", count)
	}
	if _, err := os.Stat(fullFilePath); !os.IsNotExist(err) {
		t.Fatalf("expected uploaded file to be removed, stat error=%v", err)
	}
}

func TestDeleteProgramReturnsBadRequestWhenIDFormatInvalid(t *testing.T) {
	r, token, _, _ := setupProgramCustomFieldValueTest(t)

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodDelete, "/api/programs/abc", token, nil)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestDeleteProgramReturnsNotFoundWhenMissing(t *testing.T) {
	r, token, _, _ := setupProgramCustomFieldValueTest(t)

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodDelete, "/api/programs/999", token, nil)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestGetProgramIncludesCustomFieldValuesAfterSave(t *testing.T) {
	r, token, line, program := setupProgramCustomFieldValueTest(t)
	field := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "备注", FieldType: "text", Enabled: true}
	if err := database.DB.Create(&field).Error; err != nil {
		t.Fatalf("create field: %v", err)
	}

	saveResp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, programCustomFieldValuesPath(program.ID), token, map[string]any{
		"values": []map[string]any{{"field_id": field.ID, "value": "详情页值"}},
	})
	if saveResp.Code != http.StatusOK {
		t.Fatalf("expected save status 200, got %d body=%s", saveResp.Code, saveResp.Body.String())
	}

	getResp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, programDetailPath(program.ID), token, nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("expected get status 200, got %d body=%s", getResp.Code, getResp.Body.String())
	}

	loaded := decodeProductionLineCustomFieldResponse[models.Program](t, getResp)
	if len(loaded.CustomFieldValues) != 1 {
		t.Fatalf("expected 1 custom field value, got %#v", loaded.CustomFieldValues)
	}
	if loaded.CustomFieldValues[0].ProgramID != program.ID || loaded.CustomFieldValues[0].ProductionLineCustomFieldID != field.ID || loaded.CustomFieldValues[0].Value != "详情页值" {
		t.Fatalf("unexpected custom field values: %#v", loaded.CustomFieldValues)
	}
}

func TestCreateProgramRejectsInvalidStatusAndRelations(t *testing.T) {
	r, token, line, _ := setupProgramCustomFieldValueTest(t)

	invalidStatusResp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, "/api/programs", token, map[string]any{
		"name":               "Invalid Status",
		"code":               "PROG-BAD-STATUS",
		"production_line_id": line.ID,
		"status":             "paused",
	})
	if invalidStatusResp.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid status status 400, got %d body=%s", invalidStatusResp.Code, invalidStatusResp.Body.String())
	}

	invalidLineResp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, "/api/programs", token, map[string]any{
		"name":               "Invalid Line",
		"code":               "PROG-BAD-LINE",
		"production_line_id": line.ID + 999,
		"status":             "in_progress",
	})
	if invalidLineResp.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid line status 400, got %d body=%s", invalidLineResp.Code, invalidLineResp.Body.String())
	}

	invalidVehicleResp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, "/api/programs", token, map[string]any{
		"name":               "Invalid Vehicle",
		"code":               "PROG-BAD-VEHICLE",
		"production_line_id": line.ID,
		"vehicle_model_id":   999,
		"status":             "in_progress",
	})
	if invalidVehicleResp.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid vehicle status 400, got %d body=%s", invalidVehicleResp.Code, invalidVehicleResp.Body.String())
	}
}

func TestUpdateProgramRejectsInvalidStatusAndRelations(t *testing.T) {
	r, token, _, program := setupProgramCustomFieldValueTest(t)

	invalidStatusResp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, programDetailPath(program.ID), token, map[string]any{
		"status": "paused",
	})
	if invalidStatusResp.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid status status 400, got %d body=%s", invalidStatusResp.Code, invalidStatusResp.Body.String())
	}

	invalidLineResp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, programDetailPath(program.ID), token, map[string]any{
		"production_line_id": program.ProductionLineID + 999,
	})
	if invalidLineResp.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid line status 400, got %d body=%s", invalidLineResp.Code, invalidLineResp.Body.String())
	}

	invalidVehicleResp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, programDetailPath(program.ID), token, map[string]any{
		"vehicle_model_id": 999,
	})
	if invalidVehicleResp.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid vehicle status 400, got %d body=%s", invalidVehicleResp.Code, invalidVehicleResp.Body.String())
	}
}

func TestUpdateProgramRejectsInvalidIDFormat(t *testing.T) {
	r, token, _, program := setupProgramCustomFieldValueTest(t)

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, "/api/programs/abc", token, map[string]any{
		"id":                 program.ID,
		"name":               program.Name,
		"code":               program.Code,
		"production_line_id": program.ProductionLineID,
		"vehicle_model_id":   program.VehicleModelID,
		"version":            program.Version,
		"description":        program.Description,
		"status":             program.Status,
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestUpdateProgramClearsCustomFieldValuesWhenProductionLineChanges(t *testing.T) {
	r, token, line, program := setupProgramCustomFieldValueTest(t)
	field := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "备注", FieldType: "text", Enabled: true}
	if err := database.DB.Create(&field).Error; err != nil {
		t.Fatalf("create field: %v", err)
	}
	value := models.ProgramCustomFieldValue{ProgramID: program.ID, ProductionLineCustomFieldID: field.ID, Value: "旧产线值"}
	if err := database.DB.Create(&value).Error; err != nil {
		t.Fatalf("create custom field value: %v", err)
	}

	otherLine := models.ProductionLine{Name: "产线B", Code: "LINE-002", Type: "upper", Status: "active", ProcessID: line.ProcessID}
	if err := database.DB.Create(&otherLine).Error; err != nil {
		t.Fatalf("create other line: %v", err)
	}

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, programDetailPath(program.ID), token, map[string]any{
		"id":                 program.ID,
		"name":               program.Name,
		"code":               program.Code,
		"production_line_id": otherLine.ID,
		"vehicle_model_id":   program.VehicleModelID,
		"version":            program.Version,
		"description":        program.Description,
		"status":             "in_progress",
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected update status 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var valueCount int64
	if err := database.DB.Model(&models.ProgramCustomFieldValue{}).Where("program_id = ?", program.ID).Count(&valueCount).Error; err != nil {
		t.Fatalf("count custom field values: %v", err)
	}
	if valueCount != 0 {
		t.Fatalf("expected custom field values cleared after production line change, count=%d", valueCount)
	}

	var updated models.Program
	if err := database.DB.First(&updated, program.ID).Error; err != nil {
		t.Fatalf("reload updated program: %v", err)
	}
	if updated.ProductionLineID != otherLine.ID {
		t.Fatalf("expected production line id %d, got %d", otherLine.ID, updated.ProductionLineID)
	}
}

func TestUpdateProgramAppliesCustomFieldValuesAtomically(t *testing.T) {
	r, token, line, program := setupProgramCustomFieldValueTest(t)
	validField := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "澶囨敞", FieldType: "text", Enabled: true}
	if err := database.DB.Create(&validField).Error; err != nil {
		t.Fatalf("create valid field: %v", err)
	}

	otherLine := models.ProductionLine{Name: "浜х嚎C", Code: "LINE-003", Type: "upper", Status: "active", ProcessID: line.ProcessID}
	if err := database.DB.Create(&otherLine).Error; err != nil {
		t.Fatalf("create other line: %v", err)
	}
	foreignField := models.ProductionLineCustomField{ProductionLineID: otherLine.ID, Name: "澶栭儴瀛楁", FieldType: "text", Enabled: true}
	if err := database.DB.Create(&foreignField).Error; err != nil {
		t.Fatalf("create foreign field: %v", err)
	}

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, programDetailPath(program.ID), token, map[string]any{
		"name": program.Name + "-updated",
		"custom_field_values": []map[string]any{
			{"field_id": foreignField.ID, "value": "invalid"},
		},
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected update status 400, got %d body=%s", resp.Code, resp.Body.String())
	}

	var reloaded models.Program
	if err := database.DB.First(&reloaded, program.ID).Error; err != nil {
		t.Fatalf("reload program: %v", err)
	}
	if reloaded.Name != program.Name {
		t.Fatalf("expected program name to remain %q, got %q", program.Name, reloaded.Name)
	}

	resp = performProductionLineCustomFieldRequest(t, r, http.MethodPut, programDetailPath(program.ID), token, map[string]any{
		"name": program.Name + "-updated",
		"custom_field_values": []map[string]any{
			{"field_id": validField.ID, "value": "ok"},
		},
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected update status 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var values []models.ProgramCustomFieldValue
	if err := database.DB.Where("program_id = ?", program.ID).Find(&values).Error; err != nil {
		t.Fatalf("reload custom field values: %v", err)
	}
	if len(values) != 1 || values[0].ProductionLineCustomFieldID != validField.ID || values[0].Value != "ok" {
		t.Fatalf("unexpected custom field values: %#v", values)
	}
}

func TestCreateProgramAppliesCustomFieldValuesAtomically(t *testing.T) {
	r, token, line, _ := setupProgramCustomFieldValueTest(t)
	validField := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "澶囨敞", FieldType: "text", Enabled: true}
	if err := database.DB.Create(&validField).Error; err != nil {
		t.Fatalf("create valid field: %v", err)
	}

	otherLine := models.ProductionLine{Name: "浜х嚎D", Code: "LINE-004", Type: "upper", Status: "active", ProcessID: line.ProcessID}
	if err := database.DB.Create(&otherLine).Error; err != nil {
		t.Fatalf("create other line: %v", err)
	}
	foreignField := models.ProductionLineCustomField{ProductionLineID: otherLine.ID, Name: "澶栭儴瀛楁", FieldType: "text", Enabled: true}
	if err := database.DB.Create(&foreignField).Error; err != nil {
		t.Fatalf("create foreign field: %v", err)
	}

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, "/api/programs", token, map[string]any{
		"name":               "Atomic Create",
		"code":               "ATOMIC-001",
		"production_line_id": line.ID,
		"status":             "in_progress",
		"custom_field_values": []map[string]any{
			{"field_id": foreignField.ID, "value": "invalid"},
		},
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected create status 400, got %d body=%s", resp.Code, resp.Body.String())
	}

	var count int64
	if err := database.DB.Model(&models.Program{}).Where("code = ?", "ATOMIC-001").Count(&count).Error; err != nil {
		t.Fatalf("count created programs: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected failed create to leave no program rows, count=%d", count)
	}

	resp = performProductionLineCustomFieldRequest(t, r, http.MethodPost, "/api/programs", token, map[string]any{
		"name":               "Atomic Create",
		"code":               "ATOMIC-001",
		"production_line_id": line.ID,
		"status":             "in_progress",
		"custom_field_values": []map[string]any{
			{"field_id": validField.ID, "value": "ok"},
		},
	})
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d body=%s", resp.Code, resp.Body.String())
	}

	var created models.Program
	if err := database.DB.Where("code = ?", "ATOMIC-001").First(&created).Error; err != nil {
		t.Fatalf("load created program: %v", err)
	}
	var values []models.ProgramCustomFieldValue
	if err := database.DB.Where("program_id = ?", created.ID).Find(&values).Error; err != nil {
		t.Fatalf("load created custom field values: %v", err)
	}
	if len(values) != 1 || values[0].ProductionLineCustomFieldID != validField.ID || values[0].Value != "ok" {
		t.Fatalf("unexpected created custom field values: %#v", values)
	}
}

func TestProgramMappingRejectsInvalidIDFormat(t *testing.T) {
	r, token, _, _ := setupProgramCustomFieldValueTest(t)

	parentResp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, programMappingsByParentPath("abc"), token, nil)
	if parentResp.Code != http.StatusBadRequest {
		t.Fatalf("expected parent mapping status 400, got %d body=%s", parentResp.Code, parentResp.Body.String())
	}

	childResp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, programMappingsByChildPath("abc"), token, nil)
	if childResp.Code != http.StatusBadRequest {
		t.Fatalf("expected child mapping status 400, got %d body=%s", childResp.Code, childResp.Body.String())
	}

	deleteResp := performProductionLineCustomFieldRequest(t, r, http.MethodDelete, programMappingDetailPath("abc"), token, nil)
	if deleteResp.Code != http.StatusBadRequest {
		t.Fatalf("expected delete mapping status 400, got %d body=%s", deleteResp.Code, deleteResp.Body.String())
	}
}

func TestProgramRouteIDValidation(t *testing.T) {
	r, token, _, _ := setupProgramCustomFieldValueTest(t)

	getResp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, "/api/programs/abc", token, nil)
	if getResp.Code != http.StatusBadRequest {
		t.Fatalf("expected get program status 400, got %d body=%s", getResp.Code, getResp.Body.String())
	}

	byVehicleResp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, "/api/programs/by-vehicle/abc", token, nil)
	if byVehicleResp.Code != http.StatusBadRequest {
		t.Fatalf("expected get programs by vehicle status 400, got %d body=%s", byVehicleResp.Code, byVehicleResp.Body.String())
	}

	relationsResp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, "/api/program-relations/program/abc", token, nil)
	if relationsResp.Code != http.StatusBadRequest {
		t.Fatalf("expected get relations status 400, got %d body=%s", relationsResp.Code, relationsResp.Body.String())
	}

	deleteRelationResp := performProductionLineCustomFieldRequest(t, r, http.MethodDelete, "/api/program-relations/abc", token, nil)
	if deleteRelationResp.Code != http.StatusBadRequest {
		t.Fatalf("expected delete relation status 400, got %d body=%s", deleteRelationResp.Code, deleteRelationResp.Body.String())
	}
}

func TestDeleteRelationReturnsNotFoundWhenMissing(t *testing.T) {
	r, token, _, _ := setupProgramCustomFieldValueTest(t)

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodDelete, "/api/program-relations/999", token, nil)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected delete relation status 404, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestFileAndVersionRouteIDValidation(t *testing.T) {
	r, token, _, _ := setupProgramCustomFieldValueTest(t)

	downloadResp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, "/api/files/abc/download", token, nil)
	if downloadResp.Code != http.StatusBadRequest {
		t.Fatalf("expected download file status 400, got %d body=%s", downloadResp.Code, downloadResp.Body.String())
	}

	deleteResp := performProductionLineCustomFieldRequest(t, r, http.MethodDelete, "/api/files/abc", token, nil)
	if deleteResp.Code != http.StatusBadRequest {
		t.Fatalf("expected delete file status 400, got %d body=%s", deleteResp.Code, deleteResp.Body.String())
	}

	invalidProgramIDResp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, "/api/files/program/abc", token, nil)
	if invalidProgramIDResp.Code != http.StatusBadRequest {
		t.Fatalf("expected files by program status 400, got %d body=%s", invalidProgramIDResp.Code, invalidProgramIDResp.Body.String())
	}

	missingVersionProgramIDResp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, "/api/files/download/version/v1.0.0", token, nil)
	if missingVersionProgramIDResp.Code != http.StatusBadRequest {
		t.Fatalf("expected version download missing program_id status 400, got %d body=%s", missingVersionProgramIDResp.Code, missingVersionProgramIDResp.Body.String())
	}

	invalidVersionProgramIDResp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, "/api/files/download/version/v1.0.0?program_id=abc", token, nil)
	if invalidVersionProgramIDResp.Code != http.StatusBadRequest {
		t.Fatalf("expected version download invalid program_id status 400, got %d body=%s", invalidVersionProgramIDResp.Code, invalidVersionProgramIDResp.Body.String())
	}

	blankVersionResp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, "/api/files/download/version/%20?program_id=1", token, nil)
	if blankVersionResp.Code != http.StatusBadRequest {
		t.Fatalf("expected blank version status 400, got %d body=%s", blankVersionResp.Code, blankVersionResp.Body.String())
	}

	updateVersionResp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, "/api/versions/abc", token, map[string]any{"change_log": "x"})
	if updateVersionResp.Code != http.StatusBadRequest {
		t.Fatalf("expected update version status 400, got %d body=%s", updateVersionResp.Code, updateVersionResp.Body.String())
	}

	activateVersionResp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, "/api/versions/abc/activate", token, nil)
	if activateVersionResp.Code != http.StatusBadRequest {
		t.Fatalf("expected activate version status 400, got %d body=%s", activateVersionResp.Code, activateVersionResp.Body.String())
	}
}

func TestCreateVersionRejectsMismatchedFileVersion(t *testing.T) {
	r, token, _, program := setupProgramCustomFieldValueTest(t)

	file := models.ProgramFile{
		ProgramID:   program.ID,
		FileName:    "demo.nc",
		FilePath:    "demo.nc",
		FileSize:    10,
		FileType:    ".nc",
		Version:     "v1",
		UploadedBy:  1,
		Description: "seed",
	}
	if err := database.DB.Create(&file).Error; err != nil {
		t.Fatalf("create file: %v", err)
	}

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, "/api/versions", token, map[string]any{
		"program_id": program.ID,
		"version":    "v2",
		"file_id":    file.ID,
		"change_log": "manual",
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected create version status 400, got %d body=%s", resp.Code, resp.Body.String())
	}

	var versionCount int64
	if err := database.DB.Model(&models.ProgramVersion{}).Where("program_id = ?", program.ID).Count(&versionCount).Error; err != nil {
		t.Fatalf("count versions: %v", err)
	}
	if versionCount != 0 {
		t.Fatalf("expected no version rows to be created, count=%d", versionCount)
	}
}

func TestUserRouteIDValidation(t *testing.T) {
	r, token, _, _ := setupProgramCustomFieldValueTest(t)

	tests := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{name: "get user", method: http.MethodGet, path: "/api/users/abc", body: nil},
		{name: "update user", method: http.MethodPut, path: "/api/users/abc", body: map[string]any{"name": "x"}},
		{name: "delete user", method: http.MethodDelete, path: "/api/users/abc", body: nil},
		{name: "change password", method: http.MethodPost, path: "/api/users/abc/change-password", body: map[string]any{"old_password": "x", "new_password": "y"}},
		{name: "reset password", method: http.MethodPost, path: "/api/users/abc/reset-password", body: nil},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			resp := performProductionLineCustomFieldRequest(t, r, tt.method, tt.path, token, tt.body)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
			}
		})
	}
}

func TestDeleteUserReturnsNotFoundWhenMissing(t *testing.T) {
	r, token, _, _ := setupProgramCustomFieldValueTest(t)

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodDelete, "/api/users/999", token, nil)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected delete user status 404, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestPermissionRoutesRejectInvalidUserIDFormat(t *testing.T) {
	r, token, _, _ := setupProgramCustomFieldValueTest(t)

	tests := []struct {
		name string
		path string
	}{
		{name: "user permissions", path: "/api/permissions/user/abc"},
		{name: "effective permissions", path: "/api/department-permissions/user/abc/effective"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			resp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, tt.path, token, nil)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
			}
		})
	}
}

func TestCreatePermissionUpsertsPerUserAndLineAndValidatesRelations(t *testing.T) {
	r, token, line, _ := setupProgramCustomFieldValueTest(t)

	user := models.User{
		Name:       "Permission User",
		Password:   "hashed",
		EmployeeID: "EMP-PERM-001",
		Role:       "user",
		Status:     "active",
	}
	if err := database.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	invalidResp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, "/api/permissions", token, map[string]any{
		"user_id":            user.ID + 999,
		"production_line_id": line.ID,
		"can_view":           true,
	})
	if invalidResp.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid user status 400, got %d body=%s", invalidResp.Code, invalidResp.Body.String())
	}

	createResp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, "/api/permissions", token, map[string]any{
		"user_id":            user.ID,
		"production_line_id": line.ID,
		"can_view":           true,
		"can_upload":         true,
	})
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d body=%s", createResp.Code, createResp.Body.String())
	}

	updateResp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, "/api/permissions", token, map[string]any{
		"user_id":            user.ID,
		"production_line_id": line.ID,
		"can_view":           false,
		"can_download":       true,
		"can_upload":         false,
		"can_manage":         true,
	})
	if updateResp.Code != http.StatusOK {
		t.Fatalf("expected duplicate create to update with status 200, got %d body=%s", updateResp.Code, updateResp.Body.String())
	}

	var permissions []models.UserPermission
	if err := database.DB.Where("user_id = ? AND production_line_id = ?", user.ID, line.ID).Find(&permissions).Error; err != nil {
		t.Fatalf("query permissions: %v", err)
	}
	if len(permissions) != 1 {
		t.Fatalf("expected one user-line permission row, got %#v", permissions)
	}
	if permissions[0].CanView || !permissions[0].CanDownload || permissions[0].CanUpload || !permissions[0].CanManage {
		t.Fatalf("expected permission to be updated in-place, got %#v", permissions[0])
	}
}

func TestCreateDepartmentPermissionValidatesRelations(t *testing.T) {
	r, token, line, _ := setupProgramCustomFieldValueTest(t)

	department := models.Department{Name: "Permission Department", Description: "dept", Status: "active"}
	if err := database.DB.Create(&department).Error; err != nil {
		t.Fatalf("create department: %v", err)
	}

	invalidResp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, "/api/department-permissions", token, map[string]any{
		"department_id":      department.ID + 999,
		"production_line_id": line.ID,
		"can_view":           true,
	})
	if invalidResp.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid department status 400, got %d body=%s", invalidResp.Code, invalidResp.Body.String())
	}

	createResp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, "/api/department-permissions", token, map[string]any{
		"department_id":      department.ID,
		"production_line_id": line.ID,
		"can_view":           true,
	})
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected department permission create status 201, got %d body=%s", createResp.Code, createResp.Body.String())
	}
}

func TestExportProgramsExcelReturnsAttachment(t *testing.T) {
	r, token, _, _ := setupProgramCustomFieldValueTest(t)

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, "/api/programs/export/excel", token, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	contentDisposition := resp.Header().Get("Content-Disposition")
	if contentDisposition == "" {
		t.Fatalf("expected content-disposition header, got empty")
	}
	if !strings.Contains(contentDisposition, "attachment;") {
		t.Fatalf("expected attachment in content-disposition, got %q", contentDisposition)
	}
	if !strings.Contains(contentDisposition, "filename=\"programs.xlsx\"") {
		t.Fatalf("expected ascii fallback filename in content-disposition, got %q", contentDisposition)
	}
	if !strings.Contains(contentDisposition, "filename*=UTF-8''") {
		t.Fatalf("expected RFC 5987 filename* in content-disposition, got %q", contentDisposition)
	}
	if len(resp.Body.Bytes()) < 4 {
		t.Fatalf("expected non-empty export payload")
	}
	if string(resp.Body.Bytes()[:2]) != "PK" {
		t.Fatalf("expected xlsx(zip) signature PK, got %q", string(resp.Body.Bytes()[:2]))
	}
}

func TestGetTaskStatusReturnsNotFoundAfterExpiration(t *testing.T) {
	r, token, _, _ := setupProgramCustomFieldValueTest(t)

	batchTaskMu.Lock()
	taskID := batchTaskSeq
	batchTaskSeq++
	batchTasks[taskID] = &batchImportTaskStatus{Status: "completed", ExpiresAt: time.Now().Add(-time.Second)}
	batchTaskMu.Unlock()

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, fmt.Sprintf("/api/tasks/%d/status", taskID), token, nil)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestGetTaskStatusReturnsTaskBeforeExpiration(t *testing.T) {
	r, token, _, _ := setupProgramCustomFieldValueTest(t)

	batchTaskMu.Lock()
	taskID := batchTaskSeq
	batchTaskSeq++
	batchTasks[taskID] = &batchImportTaskStatus{Status: "processing", Total: 2, Processed: 1, ExpiresAt: time.Now().Add(time.Minute)}
	batchTaskMu.Unlock()

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, fmt.Sprintf("/api/tasks/%d/status", taskID), token, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	payload := decodeProductionLineCustomFieldResponse[batchImportTaskStatus](t, resp)
	if payload.Status != "processing" {
		t.Fatalf("expected processing status, got %q", payload.Status)
	}
	if payload.Processed != 1 || payload.Total != 2 {
		t.Fatalf("unexpected task payload: %+v", payload)
	}
}

func TestGetTaskStatusConcurrentPollingIsStable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/tasks/:task_id/status", GetTaskStatus)

	batchTaskMu.Lock()
	taskID := batchTaskSeq
	batchTaskSeq++
	batchTasks[taskID] = &batchImportTaskStatus{Status: "processing", Total: 100, Processed: 10, ExpiresAt: time.Now().Add(time.Minute)}
	batchTaskMu.Unlock()

	const workers = 12
	const rounds = 20
	var wg sync.WaitGroup
	errors := make(chan string, workers*rounds)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < rounds; j++ {
				resp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, fmt.Sprintf("/api/tasks/%d/status", taskID), "", nil)
				if resp.Code != http.StatusOK {
					errors <- fmt.Sprintf("unexpected status=%d body=%s", resp.Code, resp.Body.String())
					continue
				}
				payload := decodeProductionLineCustomFieldResponse[batchImportTaskStatus](t, resp)
				if payload.Status == "" || payload.Total != 100 {
					errors <- fmt.Sprintf("unexpected payload=%+v", payload)
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Fatalf("concurrent polling failed: %s", err)
	}
}

func TestBatchImportProgramsRejectsMissingProductionLineID(t *testing.T) {
	r, token, _, _ := setupProgramCustomFieldValueTest(t)

	tmpDir := t.TempDir()
	preview := batchUploadPreview{
		TotalPrograms: 1,
		TotalFiles:    1,
		Workstations:  []batchUploadWorkstation{{Name: "WS-01", Programs: []batchUploadProgram{{Name: "P1"}}}},
	}
	previewID := createBatchPreview(preview, tmpDir, 1)

	payload := map[string]any{
		"preview_id": previewID,
		"mappings": []map[string]any{
			{"workstation_name": "WS-01", "production_line_id": nil, "vehicle_model_id": nil},
		},
	}
	resp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, "/api/batch/import", token, payload)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
	}

	body := decodeProductionLineCustomFieldResponse[map[string]any](t, resp)
	if msg, _ := body["error"].(string); !strings.Contains(msg, "production_line_id is required") {
		t.Fatalf("expected missing production_line_id error, got %v", body)
	}
}

type getProgramFilesResponse struct {
	ProgramID     uint                     `json:"program_id"`
	Versions      []getProgramFilesVersion `json:"versions"`
	TotalVersions int                      `json:"total_versions"`
	Page          int                      `json:"page"`
	PageSize      int                      `json:"page_size"`
}

type getProgramFilesVersion struct {
	ID        uint                 `json:"id"`
	Version   string               `json:"version"`
	ChangeLog string               `json:"change_log"`
	IsCurrent bool                 `json:"is_current"`
	FileCount int                  `json:"file_count"`
	Files     []models.ProgramFile `json:"files"`
}

func TestGetProgramFilesPaginatesVersionHeadersAndKeepsFileOnlyVersions(t *testing.T) {
	r, token, _, program := setupProgramCustomFieldValueTest(t)
	baseTime := time.Now().UTC().Add(-6 * time.Hour)

	seedFile := func(version, name, description string, createdAt time.Time) models.ProgramFile {
		file := models.ProgramFile{
			ProgramID:   program.ID,
			FileName:    name,
			FilePath:    name,
			FileSize:    10,
			FileType:    ".nc",
			Version:     version,
			UploadedBy:  1,
			Description: description,
			CreatedAt:   createdAt,
			UpdatedAt:   createdAt,
		}
		if err := database.DB.Create(&file).Error; err != nil {
			t.Fatalf("create file %s: %v", name, err)
		}
		return file
	}

	v1a := seedFile("v1", "v1-a.nc", "v1 first", baseTime.Add(1*time.Hour))
	v1b := seedFile("v1", "v1-b.nc", "v1 second", baseTime.Add(2*time.Hour))
	v2 := seedFile("v2", "v2-a.nc", "v2 payload", baseTime.Add(4*time.Hour))
	_ = v2
	v3 := seedFile("v3", "v3-a.nc", "orphan payload", baseTime.Add(5*time.Hour))
	_ = v3

	versions := []models.ProgramVersion{
		{
			ProgramID:  program.ID,
			Version:    "v1",
			FileID:     v1b.ID,
			UploadedBy: 1,
			ChangeLog:  "v1 log",
			IsCurrent:  false,
			CreatedAt:  baseTime.Add(3 * time.Hour),
			UpdatedAt:  baseTime.Add(3 * time.Hour),
		},
		{
			ProgramID:  program.ID,
			Version:    "v2",
			FileID:     v2.ID,
			UploadedBy: 1,
			ChangeLog:  "v2 log",
			IsCurrent:  true,
			CreatedAt:  baseTime.Add(4*time.Hour + 30*time.Minute),
			UpdatedAt:  baseTime.Add(4*time.Hour + 30*time.Minute),
		},
	}
	for _, version := range versions {
		version := version
		if err := database.DB.Create(&version).Error; err != nil {
			t.Fatalf("create version %s: %v", version.Version, err)
		}
	}

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, fmt.Sprintf("/api/files/program/%d?page=1&page_size=2", program.ID), token, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	pageOne := decodeProductionLineCustomFieldResponse[getProgramFilesResponse](t, resp)
	if pageOne.TotalVersions != 3 {
		t.Fatalf("expected 3 total versions, got %+v", pageOne)
	}
	if len(pageOne.Versions) != 2 {
		t.Fatalf("expected 2 versions on page 1, got %+v", pageOne.Versions)
	}
	if pageOne.Versions[0].Version != "v3" || pageOne.Versions[0].ID != 0 || pageOne.Versions[0].ChangeLog != "orphan payload" || pageOne.Versions[0].FileCount != 1 {
		t.Fatalf("unexpected first page version payload: %+v", pageOne.Versions[0])
	}
	if pageOne.Versions[0].IsCurrent {
		t.Fatalf("expected file-only version v3 to not be current while v2 is current: %+v", pageOne.Versions[0])
	}
	if pageOne.Versions[1].Version != "v2" || !pageOne.Versions[1].IsCurrent || pageOne.Versions[1].ChangeLog != "v2 log" || pageOne.Versions[1].FileCount != 1 {
		t.Fatalf("unexpected second page version payload: %+v", pageOne.Versions[1])
	}

	resp = performProductionLineCustomFieldRequest(t, r, http.MethodGet, fmt.Sprintf("/api/files/program/%d?page=2&page_size=2", program.ID), token, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200 on page 2, got %d body=%s", resp.Code, resp.Body.String())
	}

	pageTwo := decodeProductionLineCustomFieldResponse[getProgramFilesResponse](t, resp)
	if len(pageTwo.Versions) != 1 {
		t.Fatalf("expected 1 version on page 2, got %+v", pageTwo.Versions)
	}
	if pageTwo.Versions[0].Version != "v1" || pageTwo.Versions[0].FileCount != 2 || pageTwo.Versions[0].ChangeLog != "v1 log" {
		t.Fatalf("unexpected page 2 version payload: %+v", pageTwo.Versions[0])
	}
	if len(pageTwo.Versions[0].Files) != 2 {
		t.Fatalf("expected both v1 files on page 2, got %+v", pageTwo.Versions[0].Files)
	}
	if pageTwo.Versions[0].Files[0].ID != v1b.ID || pageTwo.Versions[0].Files[1].ID != v1a.ID {
		t.Fatalf("expected v1 files ordered by created_at desc, got %+v", pageTwo.Versions[0].Files)
	}
}

func TestGetProgramFilesMarksNewestVersionCurrentWhenVersionRowsAreMissing(t *testing.T) {
	r, token, _, program := setupProgramCustomFieldValueTest(t)
	baseTime := time.Now().UTC().Add(-2 * time.Hour)

	files := []models.ProgramFile{
		{
			ProgramID:   program.ID,
			FileName:    "older.nc",
			FilePath:    "older.nc",
			FileSize:    10,
			FileType:    ".nc",
			Version:     "older",
			UploadedBy:  1,
			Description: "older payload",
			CreatedAt:   baseTime,
			UpdatedAt:   baseTime,
		},
		{
			ProgramID:   program.ID,
			FileName:    "newer.nc",
			FilePath:    "newer.nc",
			FileSize:    10,
			FileType:    ".nc",
			Version:     "newer",
			UploadedBy:  1,
			Description: "newer payload",
			CreatedAt:   baseTime.Add(time.Hour),
			UpdatedAt:   baseTime.Add(time.Hour),
		},
	}
	for _, file := range files {
		file := file
		if err := database.DB.Create(&file).Error; err != nil {
			t.Fatalf("create file %s: %v", file.Version, err)
		}
	}

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, fmt.Sprintf("/api/files/program/%d?page=1&page_size=10", program.ID), token, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	payload := decodeProductionLineCustomFieldResponse[getProgramFilesResponse](t, resp)
	if len(payload.Versions) != 2 {
		t.Fatalf("expected 2 versions, got %+v", payload.Versions)
	}
	if payload.Versions[0].Version != "newer" || !payload.Versions[0].IsCurrent {
		t.Fatalf("expected newest file-only version to be marked current, got %+v", payload.Versions[0])
	}
	if payload.Versions[1].Version != "older" || payload.Versions[1].IsCurrent {
		t.Fatalf("expected older version to remain non-current, got %+v", payload.Versions[1])
	}
}

func TestGetProgramRelationsFiltersUnauthorizedRelatedPrograms(t *testing.T) {
	r, token, lineA, lineB := setupVehicleModelPermissionTest(t)

	allowedProgram := models.Program{
		Name:             "Allowed Relation Program",
		Code:             "PROG-REL-001",
		ProductionLineID: lineA.ID,
		Status:           "in_progress",
	}
	if err := database.DB.Create(&allowedProgram).Error; err != nil {
		t.Fatalf("create allowed program: %v", err)
	}

	blockedProgram := models.Program{
		Name:             "Blocked Relation Program",
		Code:             "PROG-REL-002",
		ProductionLineID: lineB.ID,
		Status:           "in_progress",
	}
	if err := database.DB.Create(&blockedProgram).Error; err != nil {
		t.Fatalf("create blocked program: %v", err)
	}

	visibleProgram := models.Program{
		Name:             "Visible Relation Program",
		Code:             "PROG-REL-003",
		ProductionLineID: lineA.ID,
		Status:           "in_progress",
	}
	if err := database.DB.Create(&visibleProgram).Error; err != nil {
		t.Fatalf("create visible program: %v", err)
	}

	hiddenRelation := models.ProgramRelation{
		SourceProgramID:  allowedProgram.ID,
		RelatedProgramID: blockedProgram.ID,
		RelationType:     "same_program",
	}
	if err := database.DB.Create(&hiddenRelation).Error; err != nil {
		t.Fatalf("create hidden relation: %v", err)
	}
	visibleRelation := models.ProgramRelation{
		SourceProgramID:  allowedProgram.ID,
		RelatedProgramID: visibleProgram.ID,
		RelationType:     "same_program",
	}
	if err := database.DB.Create(&visibleRelation).Error; err != nil {
		t.Fatalf("create visible relation: %v", err)
	}

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, fmt.Sprintf("/api/program-relations/program/%d", allowedProgram.ID), token, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	payload := decodeProductionLineCustomFieldResponse[[]models.ProgramRelation](t, resp)
	if len(payload) != 1 {
		t.Fatalf("expected only one visible relation, got %#v", payload)
	}
	if payload[0].ID != visibleRelation.ID {
		t.Fatalf("expected visible relation %d, got %#v", visibleRelation.ID, payload)
	}
	if payload[0].RelatedProgramID == blockedProgram.ID || payload[0].RelatedProgram.ID == blockedProgram.ID {
		t.Fatalf("expected unauthorized related program to be filtered, got %#v", payload[0])
	}
}

func TestGetVehicleModelFiltersProgramsToAuthorizedLines(t *testing.T) {
	r, token, lineA, lineB := setupVehicleModelPermissionTest(t)

	vehicleModel := models.VehicleModel{Name: "车型A", Code: "VM-001", Status: "active"}
	if err := database.DB.Create(&vehicleModel).Error; err != nil {
		t.Fatalf("create vehicle model: %v", err)
	}

	allowedProgram := models.Program{
		Name:             "授权程序",
		Code:             "PROG-VM-001",
		ProductionLineID: lineA.ID,
		VehicleModelID:   vehicleModel.ID,
		Status:           "active",
	}
	if err := database.DB.Create(&allowedProgram).Error; err != nil {
		t.Fatalf("create allowed program: %v", err)
	}

	blockedProgram := models.Program{
		Name:             "未授权程序",
		Code:             "PROG-VM-002",
		ProductionLineID: lineB.ID,
		VehicleModelID:   vehicleModel.ID,
		Status:           "active",
	}
	if err := database.DB.Create(&blockedProgram).Error; err != nil {
		t.Fatalf("create blocked program: %v", err)
	}

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, vehicleModelDetailPath(vehicleModel.ID), token, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	payload := decodeProductionLineCustomFieldResponse[models.VehicleModel](t, resp)
	if len(payload.Programs) != 1 {
		t.Fatalf("expected 1 visible program, got %#v", payload.Programs)
	}
	if payload.Programs[0].ID != allowedProgram.ID {
		t.Fatalf("expected only allowed program %d, got %#v", allowedProgram.ID, payload.Programs)
	}
	if payload.Programs[0].ProductionLineID != lineA.ID {
		t.Fatalf("expected visible program on line %d, got %#v", lineA.ID, payload.Programs[0])
	}
}

func TestGetVehicleModelsSelectorScopeIncludesUnassignedModels(t *testing.T) {
	r, token, lineA, lineB := setupVehicleModelPermissionTest(t)

	assignedModel := models.VehicleModel{Name: "?????", Code: "VM-010", Status: "active"}
	if err := database.DB.Create(&assignedModel).Error; err != nil {
		t.Fatalf("create assigned model: %v", err)
	}
	unassignedModel := models.VehicleModel{Name: "?????", Code: "VM-011", Status: "active"}
	if err := database.DB.Create(&unassignedModel).Error; err != nil {
		t.Fatalf("create unassigned model: %v", err)
	}
	forbiddenModel := models.VehicleModel{Name: "?????", Code: "VM-012", Status: "active"}
	if err := database.DB.Create(&forbiddenModel).Error; err != nil {
		t.Fatalf("create forbidden model: %v", err)
	}

	program := models.Program{
		Name:             "????",
		Code:             "PROG-VM-010",
		ProductionLineID: lineA.ID,
		VehicleModelID:   assignedModel.ID,
		Status:           "in_progress",
	}
	if err := database.DB.Create(&program).Error; err != nil {
		t.Fatalf("create program: %v", err)
	}
	forbiddenProgram := models.Program{
		Name:             "?????",
		Code:             "PROG-VM-012",
		ProductionLineID: lineB.ID,
		VehicleModelID:   forbiddenModel.ID,
		Status:           "in_progress",
	}
	if err := database.DB.Create(&forbiddenProgram).Error; err != nil {
		t.Fatalf("create forbidden program: %v", err)
	}

	defaultResp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, vehicleModelListPath(""), token, nil)
	if defaultResp.Code != http.StatusOK {
		t.Fatalf("expected default list status 200, got %d body=%s", defaultResp.Code, defaultResp.Body.String())
	}

	defaultPayload := decodeProductionLineCustomFieldResponse[[]models.VehicleModel](t, defaultResp)
	if len(defaultPayload) != 1 || defaultPayload[0].ID != assignedModel.ID {
		t.Fatalf("expected only assigned model in default list, got %#v", defaultPayload)
	}

	selectorResp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, vehicleModelListPath("selector"), token, nil)
	if selectorResp.Code != http.StatusOK {
		t.Fatalf("expected selector list status 200, got %d body=%s", selectorResp.Code, selectorResp.Body.String())
	}

	selectorPayload := decodeProductionLineCustomFieldResponse[[]models.VehicleModel](t, selectorResp)
	if len(selectorPayload) != 2 {
		t.Fatalf("expected 2 models in selector scope, got %#v", selectorPayload)
	}
	if !vehicleModelIDsContain(selectorPayload, assignedModel.ID) {
		t.Fatalf("expected selector list to include assigned model %d, got %#v", assignedModel.ID, selectorPayload)
	}
	if !vehicleModelIDsContain(selectorPayload, unassignedModel.ID) {
		t.Fatalf("expected selector list to include unassigned model %d, got %#v", unassignedModel.ID, selectorPayload)
	}
	if vehicleModelIDsContain(selectorPayload, forbiddenModel.ID) {
		t.Fatalf("expected selector list to exclude forbidden model %d, got %#v", forbiddenModel.ID, selectorPayload)
	}
}

func TestUpdateProgramAllowsClearingVehicleModelID(t *testing.T) {
	r, token, _, program := setupProgramCustomFieldValueTest(t)

	vehicleModel := models.VehicleModel{Name: "待清空车型", Code: "VM-CLEAR-001", Status: "active"}
	if err := database.DB.Create(&vehicleModel).Error; err != nil {
		t.Fatalf("create vehicle model: %v", err)
	}

	if err := database.DB.Model(&models.Program{}).Where("id = ?", program.ID).Update("vehicle_model_id", vehicleModel.ID).Error; err != nil {
		t.Fatalf("assign vehicle model: %v", err)
	}

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, programDetailPath(program.ID), token, map[string]any{
		"vehicle_model_id": nil,
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var updated models.Program
	if err := database.DB.First(&updated, program.ID).Error; err != nil {
		t.Fatalf("reload program: %v", err)
	}
	if updated.VehicleModelID != 0 {
		t.Fatalf("expected vehicle_model_id to be cleared, got %d", updated.VehicleModelID)
	}
}

func vehicleModelIDsContain(models []models.VehicleModel, targetID uint) bool {
	for _, model := range models {
		if model.ID == targetID {
			return true
		}
	}
	return false
}
