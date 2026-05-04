package services

import (
	"crane-system/database"
	"crane-system/models"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openPermissionServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Department{},
		&models.User{},
		&models.ProductionLine{},
		&models.UserPermission{},
		&models.DepartmentPermission{},
		&models.RoleDefaultPermission{},
		&models.DepartmentDefaultPermission{},
		&models.Role{},
		&models.RoleLinePermission{},
		&models.Permission{},
		&models.RolePermission{},
		&models.UserPermissionOverride{},
		&models.LineAdminAssignment{},
		&models.PermissionRule{},
	); err != nil {
		t.Fatalf("migrate service test db: %v", err)
	}
	database.DB = db
	InvalidateAllCache()
	return db
}

func TestResolveUserLinePermissionsUsesExplicitDenyBeforeDepartmentAllow(t *testing.T) {
	db := openPermissionServiceTestDB(t)
	departmentID := uint(3)
	user := models.User{ID: 9, Name: "Alice", EmployeeID: "U009", Role: "operator", Password: "x", DepartmentID: &departmentID}
	line := models.ProductionLine{ID: 21, Name: "总装线"}

	if err := db.Create(&models.Department{ID: departmentID, Name: "制造部"}).Error; err != nil {
		t.Fatalf("seed department: %v", err)
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&line).Error; err != nil {
		t.Fatalf("seed line: %v", err)
	}

	if err := SavePermissionRuleChanges(PermissionSubject{Type: models.PermissionSubjectDepartment, ID: departmentID}, []PermissionRuleChange{{
		ResourceType: models.PermissionResourceProductionLine,
		ResourceID:   line.ID,
		Action:       models.PermissionActionView,
		Decision:     models.PermissionDecisionAllow,
	}}); err != nil {
		t.Fatalf("save department allow: %v", err)
	}
	if err := SavePermissionRuleChanges(PermissionSubject{Type: models.PermissionSubjectUser, ID: user.ID}, []PermissionRuleChange{{
		ResourceType: models.PermissionResourceProductionLine,
		ResourceID:   line.ID,
		Action:       models.PermissionActionView,
		Decision:     models.PermissionDecisionDeny,
	}}); err != nil {
		t.Fatalf("save user deny: %v", err)
	}

	resolved, err := ResolveUserProductionLinePermissions(user, []models.ProductionLine{line})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	cell := resolved[0].Cells[models.PermissionActionView]
	if cell.Effective != models.PermissionDecisionDeny || cell.SourceLabel != "单独设置" {
		t.Fatalf("expected user deny, got %+v", cell)
	}
}

func TestSavePermissionRuleChangesUnsetFallsBackAndKeepsOtherCells(t *testing.T) {
	db := openPermissionServiceTestDB(t)
	departmentID := uint(4)
	user := models.User{ID: 10, Name: "Bob", EmployeeID: "U010", Role: "operator", Password: "x", DepartmentID: &departmentID}
	line := models.ProductionLine{ID: 22, Name: "调试线"}

	if err := db.Create(&models.Department{ID: departmentID, Name: "工艺部"}).Error; err != nil {
		t.Fatalf("seed department: %v", err)
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&line).Error; err != nil {
		t.Fatalf("seed line: %v", err)
	}

	if err := SavePermissionRuleChanges(PermissionSubject{Type: models.PermissionSubjectDepartment, ID: departmentID}, []PermissionRuleChange{{
		ResourceType: models.PermissionResourceProductionLine,
		ResourceID:   line.ID,
		Action:       models.PermissionActionView,
		Decision:     models.PermissionDecisionAllow,
	}}); err != nil {
		t.Fatalf("save department allow: %v", err)
	}
	if err := SavePermissionRuleChanges(PermissionSubject{Type: models.PermissionSubjectUser, ID: user.ID}, []PermissionRuleChange{
		{ResourceType: models.PermissionResourceProductionLine, ResourceID: line.ID, Action: models.PermissionActionView, Decision: models.PermissionDecisionDeny},
		{ResourceType: models.PermissionResourceProductionLine, ResourceID: line.ID, Action: models.PermissionActionDownload, Decision: models.PermissionDecisionDeny},
	}); err != nil {
		t.Fatalf("save user rules: %v", err)
	}
	if err := SavePermissionRuleChanges(PermissionSubject{Type: models.PermissionSubjectUser, ID: user.ID}, []PermissionRuleChange{{
		ResourceType: models.PermissionResourceProductionLine,
		ResourceID:   line.ID,
		Action:       models.PermissionActionView,
		Decision:     "unset",
	}}); err != nil {
		t.Fatalf("unset user view: %v", err)
	}

	resolved, err := ResolveUserProductionLinePermissions(user, []models.ProductionLine{line})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	view := resolved[0].Cells[models.PermissionActionView]
	if view.Setting != "unset" || view.Effective != models.PermissionDecisionAllow || view.SourceLabel != "部门规则" {
		t.Fatalf("expected view to fall back to department allow, got %+v", view)
	}
	download := resolved[0].Cells[models.PermissionActionDownload]
	if download.Setting != models.PermissionDecisionDeny || download.Effective != models.PermissionDecisionDeny {
		t.Fatalf("expected untouched download deny to remain, got %+v", download)
	}
}

func TestResolveUserLinePermissionsIgnoresLegacyTablesAtRuntime(t *testing.T) {
	db := openPermissionServiceTestDB(t)
	departmentID := uint(5)
	user := models.User{ID: 11, Name: "Carol", EmployeeID: "U011", Role: "operator", Password: "x", DepartmentID: &departmentID}
	line := models.ProductionLine{ID: 23, Name: "Legacy Line"}

	if err := db.Create(&models.Department{ID: departmentID, Name: "Legacy Dept"}).Error; err != nil {
		t.Fatalf("seed department: %v", err)
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&line).Error; err != nil {
		t.Fatalf("seed line: %v", err)
	}
	if err := db.Create(&models.DepartmentPermission{DepartmentID: departmentID, ProductionLineID: line.ID, CanView: true}).Error; err != nil {
		t.Fatalf("seed legacy department permission: %v", err)
	}

	resolved, err := ResolveUserProductionLinePermissions(user, []models.ProductionLine{line})
	if err != nil {
		t.Fatalf("resolve legacy-only: %v", err)
	}
	view := resolved[0].Cells[models.PermissionActionView]
	if view.Effective != models.PermissionDecisionDeny || view.Source != "system_default" {
		t.Fatalf("expected legacy table row to be ignored at runtime, got %+v", view)
	}

	if err := SavePermissionRuleChanges(PermissionSubject{Type: models.PermissionSubjectDepartment, ID: departmentID}, []PermissionRuleChange{{
		ResourceType: models.PermissionResourceProductionLine,
		ResourceID:   line.ID,
		Action:       models.PermissionActionView,
		Decision:     models.PermissionDecisionAllow,
	}}); err != nil {
		t.Fatalf("save department rule: %v", err)
	}
	resolved, err = ResolveUserProductionLinePermissions(user, []models.ProductionLine{line})
	if err != nil {
		t.Fatalf("resolve rule-backed: %v", err)
	}
	view = resolved[0].Cells[models.PermissionActionView]
	if view.Effective != models.PermissionDecisionAllow || view.Source != models.PermissionSubjectDepartment {
		t.Fatalf("expected permission_rules row to grant access, got %+v", view)
	}
}

func TestResolveUserLinePermissionsDoesNotMixRoleKeyRules(t *testing.T) {
	db := openPermissionServiceTestDB(t)
	user := models.User{ID: 12, Name: "Dave", EmployeeID: "U012", Role: "operator", Password: "x"}
	line := models.ProductionLine{ID: 24, Name: "Role Key Line"}

	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&line).Error; err != nil {
		t.Fatalf("seed line: %v", err)
	}
	if err := SavePermissionRuleChanges(PermissionSubject{Type: models.PermissionSubjectRole, Key: "viewer"}, []PermissionRuleChange{{
		ResourceType: models.PermissionResourceProductionLine,
		ResourceID:   line.ID,
		Action:       models.PermissionActionView,
		Decision:     models.PermissionDecisionAllow,
	}}); err != nil {
		t.Fatalf("save viewer rule: %v", err)
	}

	resolved, err := ResolveUserProductionLinePermissions(user, []models.ProductionLine{line})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	view := resolved[0].Cells[models.PermissionActionView]
	if view.Effective != models.PermissionDecisionDeny || view.Source != "system_default" {
		t.Fatalf("expected another role key rule to be ignored, got %+v", view)
	}
}

func TestSaveRolePermissionRuleChangesReplacesLegacyScopes(t *testing.T) {
	db := openPermissionServiceTestDB(t)
	role := models.Role{ID: 7, Name: "engineer", Description: "Engineer", Status: "active"}
	line := models.ProductionLine{ID: 31, Name: "Role Line"}

	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := db.Create(&line).Error; err != nil {
		t.Fatalf("seed line: %v", err)
	}
	legacyRules := []models.PermissionRule{
		{SubjectType: models.PermissionSubjectRole, SubjectID: role.ID, ResourceType: models.PermissionResourceProductionLine, ResourceID: line.ID, Action: models.PermissionActionView, Decision: models.PermissionDecisionAllow},
		{SubjectType: models.PermissionSubjectRole, SubjectKey: role.Name, ResourceType: models.PermissionResourceProductionLine, ResourceID: line.ID, Action: models.PermissionActionView, Decision: models.PermissionDecisionAllow},
	}
	if err := db.Create(&legacyRules).Error; err != nil {
		t.Fatalf("seed legacy role rules: %v", err)
	}

	subject := PermissionSubject{Type: models.PermissionSubjectRole, ID: role.ID, Key: role.Name}
	if err := SavePermissionRuleChanges(subject, []PermissionRuleChange{{
		ResourceType: models.PermissionResourceProductionLine,
		ResourceID:   line.ID,
		Action:       models.PermissionActionView,
		Decision:     models.PermissionDecisionDeny,
	}}); err != nil {
		t.Fatalf("save role deny: %v", err)
	}

	matrix, err := ResolveSubjectRuleMatrix(subject, []models.ProductionLine{line})
	if err != nil {
		t.Fatalf("resolve role matrix: %v", err)
	}
	view := matrix[0].Actions[models.PermissionActionView]
	if view.Setting != models.PermissionDecisionDeny || view.Effective != models.PermissionDecisionDeny {
		t.Fatalf("expected canonical role deny, got %+v", view)
	}

	var count int64
	if err := db.Model(&models.PermissionRule{}).
		Where("subject_type = ? AND resource_id = ? AND action = ?", models.PermissionSubjectRole, line.ID, models.PermissionActionView).
		Count(&count).Error; err != nil {
		t.Fatalf("count role rules: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one canonical role rule, got %d", count)
	}

	if err := SavePermissionRuleChanges(subject, []PermissionRuleChange{{
		ResourceType: models.PermissionResourceProductionLine,
		ResourceID:   line.ID,
		Action:       models.PermissionActionView,
		Decision:     "unset",
	}}); err != nil {
		t.Fatalf("unset role view: %v", err)
	}
	matrix, err = ResolveSubjectRuleMatrix(subject, []models.ProductionLine{line})
	if err != nil {
		t.Fatalf("resolve role matrix after unset: %v", err)
	}
	view = matrix[0].Actions[models.PermissionActionView]
	if view.Setting != "unset" || view.Effective != models.PermissionDecisionDeny || view.Source != "system_default" {
		t.Fatalf("expected unset to clear all role scopes, got %+v", view)
	}
}
