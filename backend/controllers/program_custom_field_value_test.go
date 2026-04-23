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

	"crane-system/database"
	"crane-system/middleware"
	"crane-system/models"
	"encoding/json"

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
		versions := api.Group("/versions")
		{
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
			permissions.GET("/user/:user_id", GetUserPermissions)
		}
		effectivePermissions := api.Group("/department-permissions")
		{
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

func TestDeleteProgramRemovesCustomFieldValues(t *testing.T) {
	r, token, line, program := setupProgramCustomFieldValueTest(t)
	field := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "备注", FieldType: "text", Enabled: true}
	if err := database.DB.Create(&field).Error; err != nil {
		t.Fatalf("create field: %v", err)
	}
	value := models.ProgramCustomFieldValue{ProgramID: program.ID, ProductionLineCustomFieldID: field.ID, Value: "待删除"}
	if err := database.DB.Create(&value).Error; err != nil {
		t.Fatalf("create value: %v", err)
	}

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodDelete, programDetailPath(program.ID), token, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var valueCount int64
	if err := database.DB.Model(&models.ProgramCustomFieldValue{}).Where("program_id = ?", program.ID).Count(&valueCount).Error; err != nil {
		t.Fatalf("count custom field values: %v", err)
	}
	if valueCount != 0 {
		t.Fatalf("expected program custom field values to be deleted, count=%d", valueCount)
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
		"status":             program.Status,
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
		TempDir:       tmpDir,
		TotalPrograms: 1,
		TotalFiles:    1,
		Workstations:  []batchUploadWorkstation{{Name: "WS-01", Programs: []batchUploadProgram{{Name: "P1"}}}},
	}
	previewBytes, err := json.Marshal(preview)
	if err != nil {
		t.Fatalf("marshal preview: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "preview.json"), previewBytes, 0o644); err != nil {
		t.Fatalf("write preview: %v", err)
	}

	payload := map[string]any{
		"temp_dir": tmpDir,
		"mappings": []map[string]any{
			{"workstation_name": "WS-01", "production_line_id": nil, "vehicle_model_id": nil},
		},
	}
	resp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, "/api/batch/import", token, payload)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
	}

	body := decodeProductionLineCustomFieldResponse[map[string]any](t, resp)
	if msg, _ := body["error"].(string); !strings.Contains(msg, "production_line_id不能为空") {
		t.Fatalf("expected missing production_line_id error, got %v", body)
	}
}
