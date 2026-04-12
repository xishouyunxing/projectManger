package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openCustomFieldTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	return db
}

func TestAutoMigrateCustomFieldModels(t *testing.T) {
	db := openCustomFieldTestDB(t)

	if err := db.AutoMigrate(
		&ProductionLine{},
		&ProductionLineCustomField{},
		&Program{},
		&ProgramCustomFieldValue{},
		&ProgramFile{},
		&ProgramVersion{},
		&ProgramRelation{},
		&VehicleModel{},
	); err != nil {
		t.Fatalf("auto migrate custom field models: %v", err)
	}

	if !db.Migrator().HasTable(&ProductionLineCustomField{}) {
		t.Fatalf("expected production_line_custom_fields table to exist")
	}

	if !db.Migrator().HasTable(&ProgramCustomFieldValue{}) {
		t.Fatalf("expected program_custom_field_values table to exist")
	}

	if !db.Migrator().HasIndex(&ProductionLineCustomField{}, "idx_production_line_custom_fields_line_name") {
		t.Fatalf("expected unique index idx_production_line_custom_fields_line_name")
	}

	if !db.Migrator().HasIndex(&ProgramCustomFieldValue{}, "idx_program_custom_field_values_program_field") {
		t.Fatalf("expected unique index idx_program_custom_field_values_program_field")
	}

	line := ProductionLine{Name: "测试产线", Code: "LINE-001", Type: "upper", Status: "active"}
	if err := db.Create(&line).Error; err != nil {
		t.Fatalf("create production line: %v", err)
	}

	field := ProductionLineCustomField{
		ProductionLineID: line.ID,
		Name:             "状态",
		FieldType:        "select",
		OptionsJSON:      `["试产","量产"]`,
		SortOrder:        1,
		Enabled:          true,
	}
	if err := db.Create(&field).Error; err != nil {
		t.Fatalf("create custom field: %v", err)
	}

	program := Program{Name: "程序A", Code: "PROG-001", ProductionLineID: line.ID, Status: "active"}
	if err := db.Create(&program).Error; err != nil {
		t.Fatalf("create program: %v", err)
	}

	value := ProgramCustomFieldValue{ProgramID: program.ID, ProductionLineCustomFieldID: field.ID, Value: "试产"}
	if err := db.Create(&value).Error; err != nil {
		t.Fatalf("create custom field value: %v", err)
	}

	var loadedLine ProductionLine
	if err := db.Preload("CustomFieldTemplates").First(&loadedLine, line.ID).Error; err != nil {
		t.Fatalf("load line with custom field templates: %v", err)
	}

	if len(loadedLine.CustomFieldTemplates) != 1 {
		t.Fatalf("expected 1 custom field template, got %d", len(loadedLine.CustomFieldTemplates))
	}

	var loadedProgram Program
	if err := db.Preload("CustomFieldValues").First(&loadedProgram, program.ID).Error; err != nil {
		t.Fatalf("load program with custom field values: %v", err)
	}

	if len(loadedProgram.CustomFieldValues) != 1 {
		t.Fatalf("expected 1 custom field value, got %d", len(loadedProgram.CustomFieldValues))
	}
}
