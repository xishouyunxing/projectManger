package database

import (
	"crane-system/config"
	"crane-system/models"
	"strings"
	"testing"
	"time"

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

func TestAutoMigrateToleratesExistingPartialVehicleModelTable(t *testing.T) {
	DB = openMigrationTestDB(t)

	if err := DB.Exec(`CREATE TABLE vehicle_models (id integer primary key, code text)`).Error; err != nil {
		t.Fatalf("create partial vehicle_models table: %v", err)
	}
	if err := DB.Exec(`CREATE UNIQUE INDEX uni_vehicle_models_code ON vehicle_models (code)`).Error; err != nil {
		t.Fatalf("create legacy unique index: %v", err)
	}

	if err := AutoMigrate(); err != nil {
		t.Fatalf("auto migrate with partial vehicle_models: %v", err)
	}

	for _, column := range []string{"name", "series", "description", "status"} {
		if !DB.Migrator().HasColumn(&models.VehicleModel{}, column) {
			t.Fatalf("expected vehicle_models.%s to be added", column)
		}
	}

	if !DB.Migrator().HasTable(&models.Program{}) {
		t.Fatal("expected later tables to still be created")
	}
}

func TestAutoMigrateSeedsBaseData(t *testing.T) {
	DB = openMigrationTestDB(t)
	previousConfig := config.AppConfig
	config.AppConfig = &config.Config{Auth: config.AuthSection{DefaultPassword: "ChangeMe123"}}
	t.Cleanup(func() { config.AppConfig = previousConfig })

	if err := AutoMigrate(); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	var roleCount int64
	if err := DB.Model(&models.Role{}).Count(&roleCount).Error; err != nil {
		t.Fatalf("count roles: %v", err)
	}
	if roleCount < 5 {
		t.Fatalf("expected preset roles, got %d", roleCount)
	}

	var permissionCount int64
	if err := DB.Model(&models.Permission{}).Count(&permissionCount).Error; err != nil {
		t.Fatalf("count permissions: %v", err)
	}
	if permissionCount < 24 {
		t.Fatalf("expected permission definitions, got %d", permissionCount)
	}

	var admin models.User
	if err := DB.Where("employee_id = ?", "admin001").First(&admin).Error; err != nil {
		t.Fatalf("find seeded admin: %v", err)
	}
	if admin.Role != "system_admin" || admin.RoleID == nil {
		t.Fatalf("expected seeded admin role to be system_admin with role_id, got role=%q role_id=%v", admin.Role, admin.RoleID)
	}

	var rolePermissionCount int64
	if err := DB.Model(&models.RolePermission{}).Count(&rolePermissionCount).Error; err != nil {
		t.Fatalf("count role permissions: %v", err)
	}
	if rolePermissionCount == 0 {
		t.Fatal("expected role permissions to be seeded")
	}

	var ruleCount int64
	if err := DB.Model(&models.PermissionRule{}).Where("resource_id = 0").Count(&ruleCount).Error; err != nil {
		t.Fatalf("count default permission rules: %v", err)
	}
	if ruleCount == 0 {
		t.Fatal("expected default permission rules to be seeded")
	}

	var migrationCount int64
	if err := DB.Model(&SchemaMigration{}).Where("name IN ?", []string{migrationStepSchemaBootstrap, migrationStepBaseSeed, migrationStepPermissionBackfill}).Count(&migrationCount).Error; err != nil {
		t.Fatalf("count migration steps: %v", err)
	}
	if migrationCount != 3 {
		t.Fatalf("expected three migration records, got %d", migrationCount)
	}
}

func TestSeedBaseDataCompletesPartialRolePermissions(t *testing.T) {
	DB = openMigrationTestDB(t)
	previousConfig := config.AppConfig
	config.AppConfig = &config.Config{Auth: config.AuthSection{DefaultPassword: "ChangeMe123"}}
	t.Cleanup(func() { config.AppConfig = previousConfig })

	if err := ensureTables(); err != nil {
		t.Fatalf("ensure tables: %v", err)
	}
	if err := seedDepartments(DB); err != nil {
		t.Fatalf("seed departments: %v", err)
	}
	if err := seedRoles(DB); err != nil {
		t.Fatalf("seed roles: %v", err)
	}
	if err := seedPermissions(DB); err != nil {
		t.Fatalf("seed permissions: %v", err)
	}

	var role models.Role
	if err := DB.Where("name = ?", "viewer").First(&role).Error; err != nil {
		t.Fatalf("find viewer role: %v", err)
	}
	var permission models.Permission
	if err := DB.Where("code = ?", "page:dashboard").First(&permission).Error; err != nil {
		t.Fatalf("find dashboard permission: %v", err)
	}
	if err := DB.Create(&models.RolePermission{RoleID: role.ID, PermissionID: permission.ID}).Error; err != nil {
		t.Fatalf("seed one role permission: %v", err)
	}

	if err := SeedBaseData(config.AppConfig); err != nil {
		t.Fatalf("seed base data: %v", err)
	}

	var count int64
	if err := DB.Model(&models.RolePermission{}).Count(&count).Error; err != nil {
		t.Fatalf("count role permissions: %v", err)
	}
	if count <= 1 {
		t.Fatalf("expected partial role permissions to be completed, got %d", count)
	}
}

func TestSeedBaseDataFillsMissingDefaultRulesWithoutOverwriting(t *testing.T) {
	DB = openMigrationTestDB(t)
	previousConfig := config.AppConfig
	config.AppConfig = &config.Config{Auth: config.AuthSection{DefaultPassword: "ChangeMe123"}}
	t.Cleanup(func() { config.AppConfig = previousConfig })

	if err := ensureTables(); err != nil {
		t.Fatalf("ensure tables: %v", err)
	}
	if err := seedDepartments(DB); err != nil {
		t.Fatalf("seed departments: %v", err)
	}
	if err := seedRoles(DB); err != nil {
		t.Fatalf("seed roles: %v", err)
	}
	if err := seedPermissions(DB); err != nil {
		t.Fatalf("seed permissions: %v", err)
	}

	var lineAdmin models.Role
	if err := DB.Where("name = ?", "line_admin").First(&lineAdmin).Error; err != nil {
		t.Fatalf("find line_admin role: %v", err)
	}
	existingRule := models.PermissionRule{
		SubjectType:  "role_default",
		SubjectID:    lineAdmin.ID,
		ResourceType: models.PermissionResourceProductionLine,
		ResourceID:   0,
		Action:       models.PermissionActionView,
		Decision:     models.PermissionDecisionDeny,
	}
	if err := DB.Create(&existingRule).Error; err != nil {
		t.Fatalf("seed existing rule: %v", err)
	}

	if err := SeedBaseData(config.AppConfig); err != nil {
		t.Fatalf("seed base data: %v", err)
	}

	var loaded models.PermissionRule
	if err := DB.Where("subject_type = ? AND subject_id = ? AND resource_id = ? AND action = ?", "role_default", lineAdmin.ID, 0, models.PermissionActionView).First(&loaded).Error; err != nil {
		t.Fatalf("load existing rule: %v", err)
	}
	if loaded.Decision != models.PermissionDecisionDeny {
		t.Fatalf("expected existing decision to stay deny, got %s", loaded.Decision)
	}

	var count int64
	if err := DB.Model(&models.PermissionRule{}).Where("subject_type = ? AND subject_id = ?", "role_default", lineAdmin.ID).Count(&count).Error; err != nil {
		t.Fatalf("count role default rules: %v", err)
	}
	if count < 4 {
		t.Fatalf("expected missing line_admin default rules to be filled, got %d", count)
	}
}

func TestAutoMigrateIsIdempotentForBaseData(t *testing.T) {
	DB = openMigrationTestDB(t)
	previousConfig := config.AppConfig
	config.AppConfig = &config.Config{Auth: config.AuthSection{DefaultPassword: "ChangeMe123"}}
	t.Cleanup(func() { config.AppConfig = previousConfig })

	if err := AutoMigrate(); err != nil {
		t.Fatalf("first auto migrate: %v", err)
	}
	var firstRolePerms int64
	var firstRules int64
	if err := DB.Model(&models.RolePermission{}).Count(&firstRolePerms).Error; err != nil {
		t.Fatalf("count first role permissions: %v", err)
	}
	if err := DB.Model(&models.PermissionRule{}).Count(&firstRules).Error; err != nil {
		t.Fatalf("count first rules: %v", err)
	}

	if err := AutoMigrate(); err != nil {
		t.Fatalf("second auto migrate: %v", err)
	}
	var secondRolePerms int64
	var secondRules int64
	if err := DB.Model(&models.RolePermission{}).Count(&secondRolePerms).Error; err != nil {
		t.Fatalf("count second role permissions: %v", err)
	}
	if err := DB.Model(&models.PermissionRule{}).Count(&secondRules).Error; err != nil {
		t.Fatalf("count second rules: %v", err)
	}

	if firstRolePerms != secondRolePerms || firstRules != secondRules {
		t.Fatalf("expected idempotent counts, role_permissions %d -> %d, rules %d -> %d", firstRolePerms, secondRolePerms, firstRules, secondRules)
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

func TestAutoMigrateSkipsLegacyBackfillAfterRecorded(t *testing.T) {
	DB = openMigrationTestDB(t)
	previousConfig := config.AppConfig
	config.AppConfig = &config.Config{Auth: config.AuthSection{DefaultPassword: "ChangeMe123"}}
	t.Cleanup(func() { config.AppConfig = previousConfig })

	if err := DB.AutoMigrate(
		&SchemaMigration{},
		&models.UserPermission{},
		&models.PermissionRule{},
	); err != nil {
		t.Fatalf("migrate initial tables: %v", err)
	}
	appliedAt := time.Now().Add(-time.Hour).UTC()
	if err := DB.Create(&SchemaMigration{Name: migrationStepPermissionBackfill, AppliedAt: appliedAt}).Error; err != nil {
		t.Fatalf("record backfill step: %v", err)
	}
	if err := DB.Model(&models.UserPermission{}).Create(map[string]any{
		"user_id":            77,
		"production_line_id": 88,
		"can_view":           true,
	}).Error; err != nil {
		t.Fatalf("seed legacy user permission: %v", err)
	}

	if err := AutoMigrate(); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	var ruleCount int64
	if err := DB.Model(&models.PermissionRule{}).Where(
		"subject_type = ? AND subject_id = ? AND resource_id = ?",
		models.PermissionSubjectUser,
		77,
		88,
	).Count(&ruleCount).Error; err != nil {
		t.Fatalf("count rules: %v", err)
	}
	if ruleCount != 0 {
		t.Fatalf("expected recorded legacy backfill to skip stale legacy rows, got %d rules", ruleCount)
	}

	var step SchemaMigration
	if err := DB.Where("name = ?", migrationStepPermissionBackfill).First(&step).Error; err != nil {
		t.Fatalf("load migration step: %v", err)
	}
	if !step.AppliedAt.Equal(appliedAt) {
		t.Fatalf("expected applied_at to remain %s, got %s", appliedAt, step.AppliedAt)
	}
}

func TestAutoMigrateDoesNotRefreshLegacyBackfillRuleTimestampsOnRepeat(t *testing.T) {
	DB = openMigrationTestDB(t)
	previousConfig := config.AppConfig
	config.AppConfig = &config.Config{Auth: config.AuthSection{DefaultPassword: "ChangeMe123"}}
	t.Cleanup(func() { config.AppConfig = previousConfig })

	if err := DB.AutoMigrate(
		&models.UserPermission{},
	); err != nil {
		t.Fatalf("migrate legacy table: %v", err)
	}
	if err := DB.Model(&models.UserPermission{}).Create(map[string]any{
		"user_id":            7,
		"production_line_id": 11,
		"can_view":           true,
	}).Error; err != nil {
		t.Fatalf("seed legacy user permission: %v", err)
	}
	if err := AutoMigrate(); err != nil {
		t.Fatalf("first auto migrate: %v", err)
	}

	var firstRule models.PermissionRule
	if err := DB.Where(
		"subject_type = ? AND subject_id = ? AND resource_id = ? AND action = ?",
		models.PermissionSubjectUser,
		7,
		11,
		models.PermissionActionView,
	).First(&firstRule).Error; err != nil {
		t.Fatalf("find backfilled rule: %v", err)
	}
	var firstStep SchemaMigration
	if err := DB.Where("name = ?", migrationStepPermissionBackfill).First(&firstStep).Error; err != nil {
		t.Fatalf("find backfill step: %v", err)
	}

	if err := AutoMigrate(); err != nil {
		t.Fatalf("second auto migrate: %v", err)
	}

	var secondRule models.PermissionRule
	if err := DB.First(&secondRule, firstRule.ID).Error; err != nil {
		t.Fatalf("reload rule: %v", err)
	}
	if !secondRule.UpdatedAt.Equal(firstRule.UpdatedAt) {
		t.Fatalf("expected repeated migrate not to refresh permission rule updated_at, %s -> %s", firstRule.UpdatedAt, secondRule.UpdatedAt)
	}
	var secondStep SchemaMigration
	if err := DB.Where("name = ?", migrationStepPermissionBackfill).First(&secondStep).Error; err != nil {
		t.Fatalf("reload backfill step: %v", err)
	}
	if !secondStep.AppliedAt.Equal(firstStep.AppliedAt) {
		t.Fatalf("expected repeated migrate not to refresh migration step, %s -> %s", firstStep.AppliedAt, secondStep.AppliedAt)
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
