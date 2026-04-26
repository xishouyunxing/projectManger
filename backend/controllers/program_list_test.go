package controllers

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	"crane-system/database"
	"crane-system/models"

	"github.com/xuri/excelize/v2"
)

func TestGetProgramsIncludesCustomFieldValueSummaries(t *testing.T) {
	r, token, line, program := setupProgramCustomFieldValueTest(t)
	enabledField := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "备注", FieldType: "text", SortOrder: 2, Enabled: true}
	if err := database.DB.Create(&enabledField).Error; err != nil {
		t.Fatalf("create enabled field: %v", err)
	}
	selectField := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "状态", FieldType: "select", SortOrder: 1, Enabled: true}
	if err := database.DB.Create(&selectField).Error; err != nil {
		t.Fatalf("create select field: %v", err)
	}
	disabledField := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "停用字段", FieldType: "text", SortOrder: 3, Enabled: true}
	if err := database.DB.Create(&disabledField).Error; err != nil {
		t.Fatalf("create disabled field: %v", err)
	}
	if err := database.DB.Model(&disabledField).Update("enabled", false).Error; err != nil {
		t.Fatalf("disable field: %v", err)
	}

	values := []models.ProgramCustomFieldValue{
		{ProgramID: program.ID, ProductionLineCustomFieldID: enabledField.ID, Value: "列表备注"},
		{ProgramID: program.ID, ProductionLineCustomFieldID: selectField.ID, Value: "量产"},
		{ProgramID: program.ID, ProductionLineCustomFieldID: disabledField.ID, Value: "不应返回"},
	}
	for _, value := range values {
		value := value
		if err := database.DB.Create(&value).Error; err != nil {
			t.Fatalf("create custom field value: %v", err)
		}
	}

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, programListPath(), token, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	type customFieldValueSummary struct {
		FieldID   uint   `json:"field_id"`
		FieldName string `json:"field_name"`
		FieldType string `json:"field_type"`
		SortOrder int    `json:"sort_order"`
		Value     string `json:"value"`
	}
	type programListItem struct {
		ID                uint                      `json:"id"`
		CustomFieldValues []customFieldValueSummary `json:"custom_field_values"`
	}

	loaded := decodeProductionLineCustomFieldResponse[[]programListItem](t, resp)
	if len(loaded) != 1 {
		t.Fatalf("expected 1 program in list, got %#v", loaded)
	}
	if loaded[0].ID != program.ID {
		t.Fatalf("expected listed program id %d, got %d", program.ID, loaded[0].ID)
	}
	if len(loaded[0].CustomFieldValues) != 2 {
		t.Fatalf("expected 2 enabled custom field summaries, got %#v", loaded[0].CustomFieldValues)
	}
	if loaded[0].CustomFieldValues[0].FieldID != selectField.ID || loaded[0].CustomFieldValues[0].FieldName != "状态" || loaded[0].CustomFieldValues[0].FieldType != "select" || loaded[0].CustomFieldValues[0].SortOrder != 1 || loaded[0].CustomFieldValues[0].Value != "量产" {
		t.Fatalf("unexpected first custom field summary: %#v", loaded[0].CustomFieldValues[0])
	}
	if loaded[0].CustomFieldValues[1].FieldID != enabledField.ID || loaded[0].CustomFieldValues[1].FieldName != "备注" || loaded[0].CustomFieldValues[1].FieldType != "text" || loaded[0].CustomFieldValues[1].SortOrder != 2 || loaded[0].CustomFieldValues[1].Value != "列表备注" {
		t.Fatalf("unexpected second custom field summary: %#v", loaded[0].CustomFieldValues[1])
	}
}

func TestGetProgramsIncludesEmptyCustomFieldValueSummaries(t *testing.T) {
	r, token, line, program := setupProgramCustomFieldValueTest(t)
	disabledField := models.ProductionLineCustomField{ProductionLineID: line.ID, Name: "停用字段", FieldType: "text", SortOrder: 1, Enabled: true}
	if err := database.DB.Create(&disabledField).Error; err != nil {
		t.Fatalf("create disabled field: %v", err)
	}
	if err := database.DB.Model(&disabledField).Update("enabled", false).Error; err != nil {
		t.Fatalf("disable field: %v", err)
	}
	value := models.ProgramCustomFieldValue{ProgramID: program.ID, ProductionLineCustomFieldID: disabledField.ID, Value: "不应返回"}
	if err := database.DB.Create(&value).Error; err != nil {
		t.Fatalf("create custom field value: %v", err)
	}

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, programListPath(), token, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	type programListItem struct {
		ID                uint             `json:"id"`
		CustomFieldValues []map[string]any `json:"custom_field_values"`
	}
	loaded := decodeProductionLineCustomFieldResponse[[]programListItem](t, resp)
	if len(loaded) != 1 {
		t.Fatalf("expected 1 program in list, got %#v", loaded)
	}
	if loaded[0].ID != program.ID {
		t.Fatalf("expected listed program id %d, got %d", program.ID, loaded[0].ID)
	}
	if loaded[0].CustomFieldValues == nil {
		t.Fatalf("expected custom_field_values to be present as empty array, got nil; body=%s", resp.Body.String())
	}
	if len(loaded[0].CustomFieldValues) != 0 {
		t.Fatalf("expected no enabled custom field summaries, got %#v", loaded[0].CustomFieldValues)
	}
}

func TestExportProgramsExcelFiltersByKeyword(t *testing.T) {
	r, token, line, _ := setupProgramCustomFieldValueTest(t)
	matchingProgram := models.Program{
		Name:             "Alpha Export Program",
		Code:             "ALPHA-001",
		ProductionLineID: line.ID,
		Status:           "active",
	}
	if err := database.DB.Create(&matchingProgram).Error; err != nil {
		t.Fatalf("create matching program: %v", err)
	}
	nonMatchingProgram := models.Program{
		Name:             "Beta Export Program",
		Code:             "BETA-001",
		ProductionLineID: line.ID,
		Status:           "active",
	}
	if err := database.DB.Create(&nonMatchingProgram).Error; err != nil {
		t.Fatalf("create non-matching program: %v", err)
	}

	resp := performProductionLineCustomFieldRequest(t, r, http.MethodGet, "/api/programs/export/excel?keyword=Alpha", token, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected export status 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	f, err := excelize.OpenReader(bytes.NewReader(resp.Body.Bytes()))
	if err != nil {
		t.Fatalf("open exported xlsx: %v", err)
	}
	defer func() { _ = f.Close() }()

	rows, err := f.GetRows("Programs")
	if err != nil {
		t.Fatalf("read exported rows: %v", err)
	}
	exportedText := ""
	for _, row := range rows {
		exportedText += strings.Join(row, "\t") + "\n"
	}

	if !strings.Contains(exportedText, matchingProgram.Name) || !strings.Contains(exportedText, matchingProgram.Code) {
		t.Fatalf("expected exported sheet to include matching program, got rows:\n%s", exportedText)
	}
	if strings.Contains(exportedText, nonMatchingProgram.Name) || strings.Contains(exportedText, nonMatchingProgram.Code) {
		t.Fatalf("expected exported sheet to exclude non-matching program, got rows:\n%s", exportedText)
	}
}
