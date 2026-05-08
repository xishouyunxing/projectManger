package services

import (
	"crane-system/models"
	"net/http"
	"testing"
)

func TestAuthorizeOwnerOrAdmin(t *testing.T) {
	if decision := AuthorizeOwnerOrAdmin(1, "system_admin", 2); !decision.Allowed {
		t.Fatalf("system_admin should be allowed, got %+v", decision)
	}

	if decision := AuthorizeOwnerOrAdmin(1, "user", 1); !decision.Allowed {
		t.Fatalf("owner should be allowed, got %+v", decision)
	}

	decision := AuthorizeOwnerOrAdmin(1, "user", 2)
	if decision.Allowed || decision.StatusCode != http.StatusForbidden {
		t.Fatalf("non-owner should be forbidden, got %+v", decision)
	}
}

func TestLinePermissionAllowsAction(t *testing.T) {
	if !LinePermissionAllowsAction(false, false, false, true, LineActionView) {
		t.Fatal("manage should imply view")
	}
	if !LinePermissionAllowsAction(false, true, false, false, LineActionDownload) {
		t.Fatal("download bit should allow download")
	}
	if LinePermissionAllowsAction(false, false, true, false, LineActionManage) {
		t.Fatal("upload should not imply manage")
	}
}

func TestCheckLineActionUsesManagedLines(t *testing.T) {
	db := openPermissionServiceTestDB(t)
	user := models.User{ID: 1, Name: "Line Admin", EmployeeID: "U001", Role: "line_admin", Password: "x"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&models.ProductionLine{ID: 2, Name: "Managed Line"}).Error; err != nil {
		t.Fatalf("seed line: %v", err)
	}
	if err := db.Create(&models.LineAdminAssignment{UserID: 1, ProductionLineID: 2}).Error; err != nil {
		t.Fatalf("seed assignment: %v", err)
	}
	InvalidateAllCache()

	if decision := CheckLineAction(1, "line_admin", 2, LineActionManage); !decision.Allowed {
		t.Fatalf("line_admin should manage assigned line, got %+v", decision)
	}

	decision := CheckLineAction(1, "user", 2, LineActionManage)
	if decision.Allowed || decision.StatusCode != http.StatusForbidden {
		t.Fatalf("ordinary user should not manage ungranted line, got %+v", decision)
	}
}
