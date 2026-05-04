package database

import (
	"crane-system/models"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	return db
}

func TestAutoMigrateCreatesMissingTables(t *testing.T) {
	DB = openMigrationTestDB(t)

	if err := AutoMigrate(); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	if !DB.Migrator().HasTable(&models.ProductionLineCustomField{}) {
		t.Fatal("expected production_line_custom_fields table to exist")
	}

	if !DB.Migrator().HasTable(&models.ProgramCustomFieldValue{}) {
		t.Fatal("expected program_custom_field_values table to exist")
	}

	if !DB.Migrator().HasTable(&models.Department{}) {
		t.Fatal("expected departments table to exist")
	}

	if !DB.Migrator().HasTable(&models.PermissionRule{}) {
		t.Fatal("expected permission_rules table to exist")
	}

	if !DB.Migrator().HasIndex(&models.PermissionRule{}, "idx_permission_rule_scope") {
		t.Fatal("expected permission_rules unique scope index to exist")
	}
}

func TestAutoMigrateBackfillsLegacyPermissionRules(t *testing.T) {
	DB = openMigrationTestDB(t)

	if err := DB.AutoMigrate(
		&models.UserPermission{},
		&models.DepartmentPermission{},
		&models.RoleDefaultPermission{},
		&models.DepartmentDefaultPermission{},
		&models.RoleLinePermission{},
	); err != nil {
		t.Fatalf("migrate legacy tables: %v", err)
	}

	if err := DB.Model(&models.UserPermission{}).Create(map[string]any{
		"user_id":            7,
		"production_line_id": 11,
		"can_view":           true,
		"can_download":       false,
		"can_upload":         false,
		"can_manage":         true,
	}).Error; err != nil {
		t.Fatalf("seed legacy user permission: %v", err)
	}

	if err := AutoMigrate(); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	if err := AutoMigrate(); err != nil {
		t.Fatalf("auto migrate second run: %v", err)
	}

	var count int64
	if err := DB.Model(&models.PermissionRule{}).Where(
		"subject_type = ? AND subject_id = ? AND resource_id = ?",
		models.PermissionSubjectUser,
		7,
		11,
	).Count(&count).Error; err != nil {
		t.Fatalf("count rules: %v", err)
	}
	if count != 4 {
		t.Fatalf("expected four action rules, got %d", count)
	}

	var download models.PermissionRule
	if err := DB.Where(
		"subject_type = ? AND subject_id = ? AND resource_id = ? AND action = ?",
		models.PermissionSubjectUser,
		7,
		11,
		models.PermissionActionDownload,
	).First(&download).Error; err != nil {
		t.Fatalf("find download rule: %v", err)
	}
	if download.Decision != models.PermissionDecisionDeny {
		t.Fatalf("expected false legacy bit to become deny, got %s", download.Decision)
	}
}

func TestValidateSchemaFailsWhenCriticalColumnMissing(t *testing.T) {
	DB = openMigrationTestDB(t)

	if err := DB.Exec(`CREATE TABLE users (id integer primary key, employee_id text)`).Error; err != nil {
		t.Fatalf("create users table: %v", err)
	}

	err := ValidateSchema()
	if err == nil {
		t.Fatal("expected validation error for missing department_id")
	}

	if !strings.Contains(err.Error(), "users") || !strings.Contains(err.Error(), "department_id") {
		t.Fatalf("unexpected error: %v", err)
	}

}

func TestValidateSchemaFailsWhenCriticalIndexMissing(t *testing.T) {
	DB = openMigrationTestDB(t)

	statements := []string{
		`CREATE TABLE users (id integer primary key, employee_id text, department_id integer)`,
		`CREATE TABLE departments (id integer primary key, name text)`,
		`CREATE TABLE production_lines (id integer primary key, code text)`,
		`CREATE TABLE vehicle_models (id integer primary key, code text)`,
		`CREATE TABLE programs (id integer primary key, production_line_id integer)`,
		`CREATE TABLE production_line_custom_fields (id integer primary key, production_line_id integer, name text)`,
		`CREATE TABLE program_custom_field_values (id integer primary key, program_id integer, production_line_custom_field_id integer)`,
	}

	for _, statement := range statements {
		if err := DB.Exec(statement).Error; err != nil {
			t.Fatalf("exec %q: %v", statement, err)
		}
	}

	err := ValidateSchema()
	if err == nil {
		t.Fatal("expected validation error for missing critical index")
	}

	if !strings.Contains(err.Error(), "idx_production_line_custom_fields_line_name") {
		t.Fatalf("unexpected error: %v", err)
	}
}
