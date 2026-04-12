package controllers

import (
	"fmt"
	"net/http"
	"testing"

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
			programs.GET("/:id", GetProgram)
			programs.PUT("/:id", UpdateProgram)
			programs.PUT("/:id/custom-field-values", SaveProgramCustomFieldValues)
			programs.DELETE("/:id", DeleteProgram)
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
