package controllers

import (
	"bytes"
	"crane-system/config"
	"crane-system/database"
	"crane-system/middleware"
	"crane-system/models"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openProductionLineCustomFieldTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	config.LoadConfig()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Process{},
		&models.ProductionLine{},
		&models.ProductionLineCustomField{},
		&models.Program{},
		&models.ProgramCustomFieldValue{},
		&models.ProgramFile{},
		&models.ProgramVersion{},
		&models.ProgramRelation{},
		&models.UserPermission{},
		&models.VehicleModel{},
	); err != nil {
		t.Fatalf("auto migrate test db: %v", err)
	}

	return db
}

func createProductionLineCustomFieldAdminToken(t *testing.T, userID uint) string {
	t.Helper()

	claims := &middleware.Claims{
		UserID: userID,
		Role:   "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.AppConfig.Auth.JWTSecret))
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}

	return tokenString
}

func seedProductionLineCustomFieldAuthData(t *testing.T, db *gorm.DB) (string, models.ProductionLine) {
	t.Helper()

	admin := models.User{
		Name:       "Admin",
		Password:   "hashed",
		EmployeeID: "EMP-001",
		Role:       "admin",
		Status:     "active",
	}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	process := models.Process{Name: "总装", Code: "PROC-001", Type: "upper"}
	if err := db.Create(&process).Error; err != nil {
		t.Fatalf("create process: %v", err)
	}

	line := models.ProductionLine{Name: "产线A", Code: "LINE-001", Type: "upper", Status: "active", ProcessID: process.ID}
	if err := db.Create(&line).Error; err != nil {
		t.Fatalf("create production line: %v", err)
	}

	return createProductionLineCustomFieldAdminToken(t, admin.ID), line
}

func setupProductionLineCustomFieldTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		lines := api.Group("/production-lines")
		{
			lines.GET("/:id/custom-fields", GetProductionLineCustomFields)
			lines.POST("/:id/custom-fields", middleware.AdminMiddleware(), CreateProductionLineCustomField)
			lines.PUT("/:id/custom-fields/:fieldId", middleware.AdminMiddleware(), UpdateProductionLineCustomField)
			lines.DELETE("/:id/custom-fields/:fieldId", middleware.AdminMiddleware(), DeleteProductionLineCustomField)
		}
	}

	return r
}

func performProductionLineCustomFieldRequest(t *testing.T, r http.Handler, method, path string, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var requestBody []byte
	if body != nil {
		var err error
		requestBody, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(requestBody))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func decodeProductionLineCustomFieldResponse[T any](t *testing.T, resp *httptest.ResponseRecorder) T {
	t.Helper()

	var payload T
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, resp.Body.String())
	}

	return payload
}

func productionLineCustomFieldPath(lineID uint) string {
	return fmt.Sprintf("/api/production-lines/%d/custom-fields", lineID)
}

func productionLineCustomFieldDetailPath(lineID, fieldID uint) string {
	return fmt.Sprintf("/api/production-lines/%d/custom-fields/%d", lineID, fieldID)
}

func setupProductionLineCustomFieldCreateTest(t *testing.T) (*gin.Engine, string, models.ProductionLine) {
	t.Helper()
	database.DB = openProductionLineCustomFieldTestDB(t)
	token, line := seedProductionLineCustomFieldAuthData(t, database.DB)
	return setupProductionLineCustomFieldTestRouter(), token, line
}

func setupProductionLineCustomFieldUpdateTest(t *testing.T) (*gin.Engine, string, models.ProductionLine, models.ProductionLine, models.ProductionLineCustomField, models.ProductionLineCustomField) {
	t.Helper()
	database.DB = openProductionLineCustomFieldTestDB(t)
	token, line := seedProductionLineCustomFieldAuthData(t, database.DB)

	field := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "状态", FieldType: "text", SortOrder: 1, Enabled: true}
	if err := database.DB.Create(&field).Error; err != nil {
		t.Fatalf("create field: %v", err)
	}
	otherLine := models.ProductionLine{Name: "产线B", Code: "LINE-002", Type: "upper", Status: "active", ProcessID: line.ProcessID}
	if err := database.DB.Create(&otherLine).Error; err != nil {
		t.Fatalf("create other line: %v", err)
	}
	otherField := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "阶段", FieldType: "select", OptionsJSON: `["试产"]`, SortOrder: 2, Enabled: true}
	if err := database.DB.Create(&otherField).Error; err != nil {
		t.Fatalf("create other field: %v", err)
	}

	return setupProductionLineCustomFieldTestRouter(), token, line, otherLine, field, otherField
}

func TestGetProductionLineCustomFieldsSortsBySortOrderThenID(t *testing.T) {
	database.DB = openProductionLineCustomFieldTestDB(t)
	token, line := seedProductionLineCustomFieldAuthData(t, database.DB)

	fields := []models.ProductionLineCustomField{
		{ProductionLineID: line.ID, Name: "后装", FieldType: "text", SortOrder: 2, Enabled: true},
		{ProductionLineID: line.ID, Name: "中装", FieldType: "text", SortOrder: 1, Enabled: true},
		{ProductionLineID: line.ID, Name: "前装", FieldType: "text", SortOrder: 1, Enabled: true},
	}
	for _, field := range fields {
		field := field
		if err := database.DB.Create(&field).Error; err != nil {
			t.Fatalf("create field: %v", err)
		}
	}

	r := setupProductionLineCustomFieldTestRouter()
	resp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, productionLineCustomFieldPath(line.ID), token, nil)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var payload []models.ProductionLineCustomField
	payload = decodeProductionLineCustomFieldResponse[[]models.ProductionLineCustomField](t, resp)
	if len(payload) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(payload))
	}

	if payload[0].Name != "中装" || payload[1].Name != "前装" || payload[2].Name != "后装" {
		t.Fatalf("unexpected order: %#v", payload)
	}
	if payload[0].ID >= payload[1].ID {
		t.Fatalf("expected same sort_order fields ordered by id asc, got ids %d and %d", payload[0].ID, payload[1].ID)
	}
}

func TestCreateProductionLineCustomFieldValidatesAndPersists(t *testing.T) {
	t.Run("creates select field with options", func(t *testing.T) {
		r, token, line := setupProductionLineCustomFieldCreateTest(t)
		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, productionLineCustomFieldPath(line.ID), token, map[string]any{
			"name":         "状态",
			"field_type":   "select",
			"options_json": `["试产","量产"]`,
			"sort_order":   3,
			"enabled":      true,
		})

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d body=%s", resp.Code, resp.Body.String())
		}

		field := decodeProductionLineCustomFieldResponse[models.ProductionLineCustomField](t, resp)
		if field.ProductionLineID != line.ID || field.Name != "状态" || field.FieldType != "select" || field.OptionsJSON != `["试产","量产"]` || field.SortOrder != 3 || !field.Enabled {
			t.Fatalf("unexpected field payload: %#v", field)
		}
	})

	t.Run("defaults enabled to true when omitted", func(t *testing.T) {
		r, token, line := setupProductionLineCustomFieldCreateTest(t)
		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, productionLineCustomFieldPath(line.ID), token, map[string]any{
			"name":       "班次",
			"field_type": "text",
		})

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d body=%s", resp.Code, resp.Body.String())
		}

		field := decodeProductionLineCustomFieldResponse[models.ProductionLineCustomField](t, resp)
		if !field.Enabled {
			t.Fatalf("expected enabled default true, got %#v", field)
		}
	})

	t.Run("text field ignores options json", func(t *testing.T) {
		r, token, line := setupProductionLineCustomFieldCreateTest(t)
		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, productionLineCustomFieldPath(line.ID), token, map[string]any{
			"name":         "备注",
			"field_type":   "text",
			"options_json": `["should","clear"]`,
		})

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d body=%s", resp.Code, resp.Body.String())
		}

		field := decodeProductionLineCustomFieldResponse[models.ProductionLineCustomField](t, resp)
		if field.OptionsJSON != "" {
			t.Fatalf("expected text field options_json to be empty, got %#v", field)
		}
	})

	t.Run("rejects duplicate names in same line", func(t *testing.T) {
		r, token, line := setupProductionLineCustomFieldCreateTest(t)
		existing := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "状态", FieldType: "text", Enabled: true}
		if err := database.DB.Create(&existing).Error; err != nil {
			t.Fatalf("create existing field: %v", err)
		}

		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, productionLineCustomFieldPath(line.ID), token, map[string]any{
			"name":       "状态",
			"field_type": "text",
		})

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("allows same name on different line", func(t *testing.T) {
		r, token, line := setupProductionLineCustomFieldCreateTest(t)
		existing := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "状态", FieldType: "text", Enabled: true}
		if err := database.DB.Create(&existing).Error; err != nil {
			t.Fatalf("create existing field: %v", err)
		}

		otherLine := models.ProductionLine{Name: "产线B", Code: "LINE-002", Type: "upper", Status: "active", ProcessID: line.ProcessID}
		if err := database.DB.Create(&otherLine).Error; err != nil {
			t.Fatalf("create other line: %v", err)
		}

		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, productionLineCustomFieldPath(otherLine.ID), token, map[string]any{
			"name":       "状态",
			"field_type": "text",
		})

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("rejects unsupported field type", func(t *testing.T) {
		r, token, line := setupProductionLineCustomFieldCreateTest(t)
		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, productionLineCustomFieldPath(line.ID), token, map[string]any{
			"name":       "优先级",
			"field_type": "number",
		})

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("rejects select field without valid options", func(t *testing.T) {
		r, token, line := setupProductionLineCustomFieldCreateTest(t)
		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, productionLineCustomFieldPath(line.ID), token, map[string]any{
			"name":         "阶段",
			"field_type":   "select",
			"options_json": `[]`,
		})

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("rejects missing production line", func(t *testing.T) {
		r, token, _ := setupProductionLineCustomFieldCreateTest(t)
		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, "/api/production-lines/999/custom-fields", token, map[string]any{
			"name":       "状态",
			"field_type": "text",
		})

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("returns internal error when uniqueness check fails", func(t *testing.T) {
		r, token, line := setupProductionLineCustomFieldCreateTest(t)
		if err := database.DB.Migrator().DropTable(&models.ProductionLineCustomField{}); err != nil {
			t.Fatalf("drop custom field table: %v", err)
		}

		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPost, productionLineCustomFieldPath(line.ID), token, map[string]any{
			"name":       "状态",
			"field_type": "text",
		})

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d body=%s", resp.Code, resp.Body.String())
		}
	})
}

func TestUpdateProductionLineCustomFieldValidatesLineOwnershipAndPayload(t *testing.T) {
	t.Run("updates field within same line", func(t *testing.T) {
		r, token, line, _, field, _ := setupProductionLineCustomFieldUpdateTest(t)
		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, productionLineCustomFieldDetailPath(line.ID, field.ID), token, map[string]any{
			"name":         "生产状态",
			"field_type":   "select",
			"options_json": `["试产","量产"]`,
			"sort_order":   5,
			"enabled":      false,
		})

		if resp.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d body=%s", resp.Code, resp.Body.String())
		}

		updated := decodeProductionLineCustomFieldResponse[models.ProductionLineCustomField](t, resp)
		if updated.ID != field.ID || updated.ProductionLineID != line.ID || updated.Name != "生产状态" || updated.FieldType != "select" || updated.OptionsJSON != `["试产","量产"]` || updated.SortOrder != 5 || updated.Enabled {
			t.Fatalf("unexpected field payload: %#v", updated)
		}
	})

	t.Run("preserves omitted optional fields on update", func(t *testing.T) {
		r, token, line, _, field, _ := setupProductionLineCustomFieldUpdateTest(t)
		field.FieldType = "select"
		field.OptionsJSON = `["试产","量产"]`
		field.SortOrder = 5
		field.Enabled = false
		if err := database.DB.Save(&field).Error; err != nil {
			t.Fatalf("prepare field for partial update: %v", err)
		}

		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, productionLineCustomFieldDetailPath(line.ID, field.ID), token, map[string]any{
			"name": "保留字段",
		})

		if resp.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d body=%s", resp.Code, resp.Body.String())
		}

		updated := decodeProductionLineCustomFieldResponse[models.ProductionLineCustomField](t, resp)
		if updated.Name != "保留字段" {
			t.Fatalf("expected name update, got %#v", updated)
		}
		if updated.FieldType != "select" || updated.OptionsJSON != `["试产","量产"]` || updated.SortOrder != 5 || updated.Enabled {
			t.Fatalf("expected omitted fields preserved, got %#v", updated)
		}
	})

	t.Run("text update clears options json", func(t *testing.T) {
		r, token, line, _, field, _ := setupProductionLineCustomFieldUpdateTest(t)
		field.FieldType = "select"
		field.OptionsJSON = `["试产"]`
		if err := database.DB.Save(&field).Error; err != nil {
			t.Fatalf("prepare field for text update: %v", err)
		}

		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, productionLineCustomFieldDetailPath(line.ID, field.ID), token, map[string]any{
			"field_type":   "text",
			"options_json": `["ignored"]`,
		})

		if resp.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d body=%s", resp.Code, resp.Body.String())
		}

		updated := decodeProductionLineCustomFieldResponse[models.ProductionLineCustomField](t, resp)
		if updated.FieldType != "text" || updated.OptionsJSON != "" {
			t.Fatalf("expected text field with cleared options, got %#v", updated)
		}
	})

	t.Run("rejects duplicate names in same line", func(t *testing.T) {
		r, token, line, _, field, _ := setupProductionLineCustomFieldUpdateTest(t)
		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, productionLineCustomFieldDetailPath(line.ID, field.ID), token, map[string]any{
			"name":       "阶段",
			"field_type": "text",
		})

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("allows updating same name on current record", func(t *testing.T) {
		r, token, line, _, field, _ := setupProductionLineCustomFieldUpdateTest(t)
		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, productionLineCustomFieldDetailPath(line.ID, field.ID), token, map[string]any{
			"name":       "状态",
			"sort_order": 9,
		})

		if resp.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("rejects field from another line", func(t *testing.T) {
		r, token, _, otherLine, field, _ := setupProductionLineCustomFieldUpdateTest(t)
		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, productionLineCustomFieldDetailPath(otherLine.ID, field.ID), token, map[string]any{
			"name":       "状态",
			"field_type": "text",
		})

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("rejects invalid select options", func(t *testing.T) {
		r, token, line, _, field, _ := setupProductionLineCustomFieldUpdateTest(t)
		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, productionLineCustomFieldDetailPath(line.ID, field.ID), token, map[string]any{
			"name":         "状态",
			"field_type":   "select",
			"options_json": `[""]`,
		})

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("returns internal error when uniqueness check fails", func(t *testing.T) {
		r, token, line, _, field, _ := setupProductionLineCustomFieldUpdateTest(t)
		if err := database.DB.Migrator().DropTable(&models.ProductionLineCustomField{}); err != nil {
			t.Fatalf("drop custom field table: %v", err)
		}

		resp := performProductionLineCustomFieldRequest(t, r, http.MethodPut, productionLineCustomFieldDetailPath(line.ID, field.ID), token, map[string]any{
			"name": "新名称",
		})

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d body=%s", resp.Code, resp.Body.String())
		}
	})
}

func TestDeleteProductionLineCustomFieldRequiresLineOwnership(t *testing.T) {
	t.Run("deletes field within same line", func(t *testing.T) {
		database.DB = openProductionLineCustomFieldTestDB(t)
		token, line := seedProductionLineCustomFieldAuthData(t, database.DB)
		field := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "状态", FieldType: "text", Enabled: true}
		if err := database.DB.Create(&field).Error; err != nil {
			t.Fatalf("create field: %v", err)
		}

		r := setupProductionLineCustomFieldTestRouter()
		resp := performProductionLineCustomFieldRequest(t, r, http.MethodDelete, productionLineCustomFieldDetailPath(line.ID, field.ID), token, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d body=%s", resp.Code, resp.Body.String())
		}

		var count int64
		if err := database.DB.Model(&models.ProductionLineCustomField{}).Where("id = ?", field.ID).Count(&count).Error; err != nil {
			t.Fatalf("count field: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected field to be deleted, count=%d", count)
		}
	})

	t.Run("rejects field from another line", func(t *testing.T) {
		database.DB = openProductionLineCustomFieldTestDB(t)
		token, line := seedProductionLineCustomFieldAuthData(t, database.DB)
		field := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "状态", FieldType: "text", Enabled: true}
		if err := database.DB.Create(&field).Error; err != nil {
			t.Fatalf("create field: %v", err)
		}
		otherLine := models.ProductionLine{Name: "产线B", Code: "LINE-002", Type: "upper", Status: "active", ProcessID: line.ProcessID}
		if err := database.DB.Create(&otherLine).Error; err != nil {
			t.Fatalf("create other line: %v", err)
		}

		r := setupProductionLineCustomFieldTestRouter()
		resp := performProductionLineCustomFieldRequest(t, r, http.MethodDelete, productionLineCustomFieldDetailPath(otherLine.ID, field.ID), token, nil)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("returns line not found when production line is missing", func(t *testing.T) {
		database.DB = openProductionLineCustomFieldTestDB(t)
		token, line := seedProductionLineCustomFieldAuthData(t, database.DB)
		field := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "状态", FieldType: "text", Enabled: true}
		if err := database.DB.Create(&field).Error; err != nil {
			t.Fatalf("create field: %v", err)
		}
		if err := database.DB.Delete(&models.ProductionLine{}, line.ID).Error; err != nil {
			t.Fatalf("delete production line: %v", err)
		}

		r := setupProductionLineCustomFieldTestRouter()
		resp := performProductionLineCustomFieldRequest(t, r, http.MethodDelete, productionLineCustomFieldDetailPath(line.ID, field.ID), token, nil)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d body=%s", resp.Code, resp.Body.String())
		}

		payload := decodeProductionLineCustomFieldResponse[map[string]string](t, resp)
		if payload["error"] != "生产线不存在" {
			t.Fatalf("expected production line not found error, got %#v", payload)
		}
	})
}

func TestDeleteProductionLineCustomFieldBlocksWhenProgramValuesExist(t *testing.T) {
	database.DB = openProductionLineCustomFieldTestDB(t)
	token, line := seedProductionLineCustomFieldAuthData(t, database.DB)

	field := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "状态", FieldType: "text", Enabled: true}
	if err := database.DB.Create(&field).Error; err != nil {
		t.Fatalf("create field: %v", err)
	}
	program := models.Program{Name: "程序A", Code: "PROG-001", ProductionLineID: line.ID, Status: "active"}
	if err := database.DB.Create(&program).Error; err != nil {
		t.Fatalf("create program: %v", err)
	}
	value := models.ProgramCustomFieldValue{ProgramID: program.ID, ProductionLineCustomFieldID: field.ID, Value: "试产"}
	if err := database.DB.Create(&value).Error; err != nil {
		t.Fatalf("create value: %v", err)
	}

	r := setupProductionLineCustomFieldTestRouter()
	resp := performProductionLineCustomFieldRequest(t, r, http.MethodDelete, productionLineCustomFieldDetailPath(line.ID, field.ID), token, nil)

	if resp.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d body=%s", resp.Code, resp.Body.String())
	}

	var fieldCount int64
	if err := database.DB.Model(&models.ProductionLineCustomField{}).Where("id = ?", field.ID).Count(&fieldCount).Error; err != nil {
		t.Fatalf("count field: %v", err)
	}
	if fieldCount != 1 {
		t.Fatalf("expected field to remain, count=%d", fieldCount)
	}

	var valueCount int64
	if err := database.DB.Model(&models.ProgramCustomFieldValue{}).Where("id = ?", value.ID).Count(&valueCount).Error; err != nil {
		t.Fatalf("count value: %v", err)
	}
	if valueCount != 1 {
		t.Fatalf("expected value to remain, count=%d", valueCount)
	}
}
