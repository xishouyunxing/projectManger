package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"crane-system/services"
	"errors"
	"testing"

	"gorm.io/gorm"
)

func TestLoadLegacyUserPermissionBySyntheticID(t *testing.T) {
	database.DB = openProductionLineCustomFieldTestDB(t)
	services.InvalidateAllCache()

	user := models.User{Name: "Legacy User", EmployeeID: "LEG-U", Role: "user", Password: "x", Status: "active"}
	line := models.ProductionLine{Name: "Legacy User Line", Code: "LEG-U-LINE", Type: "upper", Status: "active"}
	if err := database.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := database.DB.Create(&line).Error; err != nil {
		t.Fatalf("create line: %v", err)
	}
	if err := services.SavePermissionRuleChanges(services.PermissionSubject{Type: models.PermissionSubjectUser, ID: user.ID}, []services.PermissionRuleChange{
		{ResourceType: models.PermissionResourceProductionLine, ResourceID: line.ID, Action: models.PermissionActionView, Decision: models.PermissionDecisionAllow},
		{ResourceType: models.PermissionResourceProductionLine, ResourceID: line.ID, Action: models.PermissionActionDownload, Decision: models.PermissionDecisionDeny},
	}); err != nil {
		t.Fatalf("save user rules: %v", err)
	}

	permissionID := services.SyntheticLinePermissionID(user.ID, line.ID)
	var permission models.UserPermission
	if err := loadLegacyUserPermissionByID(permissionID, &permission); err != nil {
		t.Fatalf("load legacy user permission: %v", err)
	}
	if permission.UserID != user.ID || permission.ProductionLineID != line.ID || !permission.CanView || permission.CanDownload {
		t.Fatalf("unexpected legacy user permission: %+v", permission)
	}

	for _, invalidID := range []uint{
		0,
		services.SyntheticLinePermissionID(user.ID+100, line.ID),
		services.SyntheticLinePermissionID(user.ID, line.ID+100),
	} {
		if err := loadLegacyUserPermissionByID(invalidID, &permission); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("expected not found for user permission id %d, got %v", invalidID, err)
		}
	}
}

func TestLoadLegacyDepartmentPermissionBySyntheticID(t *testing.T) {
	database.DB = openProductionLineCustomFieldTestDB(t)
	services.InvalidateAllCache()

	department := models.Department{Name: "Legacy Dept"}
	line := models.ProductionLine{Name: "Legacy Dept Line", Code: "LEG-D-LINE", Type: "upper", Status: "active"}
	if err := database.DB.Create(&department).Error; err != nil {
		t.Fatalf("create department: %v", err)
	}
	if err := database.DB.Create(&line).Error; err != nil {
		t.Fatalf("create line: %v", err)
	}
	if err := services.SavePermissionRuleChanges(services.PermissionSubject{Type: models.PermissionSubjectDepartment, ID: department.ID}, []services.PermissionRuleChange{
		{ResourceType: models.PermissionResourceProductionLine, ResourceID: line.ID, Action: models.PermissionActionUpload, Decision: models.PermissionDecisionAllow},
		{ResourceType: models.PermissionResourceProductionLine, ResourceID: line.ID, Action: models.PermissionActionManage, Decision: models.PermissionDecisionDeny},
	}); err != nil {
		t.Fatalf("save department rules: %v", err)
	}

	permissionID := services.SyntheticLinePermissionID(department.ID, line.ID)
	var permission models.DepartmentPermission
	if err := loadLegacyDepartmentPermissionByID(permissionID, &permission); err != nil {
		t.Fatalf("load legacy department permission: %v", err)
	}
	if permission.DepartmentID != department.ID || permission.ProductionLineID != line.ID || !permission.CanUpload || permission.CanManage {
		t.Fatalf("unexpected legacy department permission: %+v", permission)
	}

	for _, invalidID := range []uint{
		0,
		services.SyntheticLinePermissionID(department.ID+100, line.ID),
		services.SyntheticLinePermissionID(department.ID, line.ID+100),
	} {
		if err := loadLegacyDepartmentPermissionByID(invalidID, &permission); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("expected not found for department permission id %d, got %v", invalidID, err)
		}
	}
}
