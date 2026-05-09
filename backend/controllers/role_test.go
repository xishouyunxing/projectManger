package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"crane-system/services"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDeleteRoleRemovesRoleKeyDefaultPermissionRules(t *testing.T) {
	database.DB = openProductionLineCustomFieldTestDB(t)
	services.InvalidateAllCache()

	role := models.Role{Name: "custom_reviewer", Description: "Custom Reviewer", Status: "active"}
	if err := database.DB.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	line := models.ProductionLine{Name: "Delete Role Line", Code: "DEL-ROLE", Type: "upper", Status: "active"}
	if err := database.DB.Create(&line).Error; err != nil {
		t.Fatalf("create line: %v", err)
	}
	permission := models.Permission{Name: "View", Code: "page:view", Type: "page", Resource: "program"}
	if err := database.DB.Create(&permission).Error; err != nil {
		t.Fatalf("create permission: %v", err)
	}
	if err := database.DB.Create(&models.RolePermission{RoleID: role.ID, PermissionID: permission.ID}).Error; err != nil {
		t.Fatalf("create role permission: %v", err)
	}
	if err := database.DB.Create(&models.RoleLinePermission{RoleID: role.ID, ProductionLineID: line.ID, CanView: true}).Error; err != nil {
		t.Fatalf("create role line permission: %v", err)
	}
	rules := []models.PermissionRule{
		{
			SubjectType:  models.PermissionSubjectRole,
			SubjectID:    role.ID,
			ResourceType: models.PermissionResourceProductionLine,
			ResourceID:   line.ID,
			Action:       models.PermissionActionView,
			Decision:     models.PermissionDecisionAllow,
		},
		{
			SubjectType:  models.PermissionSubjectRole,
			SubjectID:    0,
			SubjectKey:   role.Name,
			ResourceType: models.PermissionResourceProductionLine,
			ResourceID:   line.ID,
			Action:       models.PermissionActionDownload,
			Decision:     models.PermissionDecisionAllow,
		},
		{
			SubjectType:  "role_default",
			SubjectID:    role.ID,
			ResourceType: models.PermissionResourceProductionLine,
			ResourceID:   line.ID,
			Action:       models.PermissionActionUpload,
			Decision:     models.PermissionDecisionAllow,
		},
	}
	if err := database.DB.Create(&rules).Error; err != nil {
		t.Fatalf("create permission rules: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.DELETE("/roles/:id", DeleteRole)

	req := httptest.NewRequest(http.MethodDelete, "/roles/1", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var roleCount int64
	if err := database.DB.Unscoped().Model(&models.Role{}).Where("id = ? AND deleted_at IS NULL", role.ID).Count(&roleCount).Error; err != nil {
		t.Fatalf("count role: %v", err)
	}
	if roleCount != 0 {
		t.Fatalf("expected role to be deleted, got %d", roleCount)
	}

	var ruleCount int64
	if err := database.DB.Unscoped().Model(&models.PermissionRule{}).
		Where("(subject_type = ? AND subject_id = ?) OR (subject_type = ? AND subject_id = ? AND subject_key = ?) OR (subject_type = ? AND subject_id = ?)",
			models.PermissionSubjectRole, role.ID,
			models.PermissionSubjectRole, 0, role.Name,
			"role_default", role.ID,
		).
		Count(&ruleCount).Error; err != nil {
		t.Fatalf("count permission rules: %v", err)
	}
	if ruleCount != 0 {
		t.Fatalf("expected role permission rules to be hard deleted, got %d", ruleCount)
	}

	var rolePermissionCount int64
	if err := database.DB.Model(&models.RolePermission{}).Where("role_id = ?", role.ID).Count(&rolePermissionCount).Error; err != nil {
		t.Fatalf("count role permissions: %v", err)
	}
	if rolePermissionCount != 0 {
		t.Fatalf("expected role permissions to be deleted, got %d", rolePermissionCount)
	}

	var roleLinePermissionCount int64
	if err := database.DB.Model(&models.RoleLinePermission{}).Where("role_id = ?", role.ID).Count(&roleLinePermissionCount).Error; err != nil {
		t.Fatalf("count role line permissions: %v", err)
	}
	if roleLinePermissionCount != 0 {
		t.Fatalf("expected role line permissions to be deleted, got %d", roleLinePermissionCount)
	}
}
