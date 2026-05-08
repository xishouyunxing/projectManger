package controllers

import (
	"bytes"
	"crane-system/config"
	"crane-system/database"
	"crane-system/middleware"
	"crane-system/models"
	"crane-system/services"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type lineAdminSecurityFixture struct {
	Router      *gin.Engine
	Line        models.ProductionLine
	OtherLine   models.ProductionLine
	TargetUser  models.User
	NormalToken string
	LineToken   string
	AdminToken  string
	PermToken   string
}

func setupLineAdminSecurityFixture(t *testing.T) lineAdminSecurityFixture {
	t.Helper()

	database.DB = openProductionLineCustomFieldTestDB(t)
	if config.AppConfig == nil {
		t.Fatal("expected test config to be loaded")
	}
	config.AppConfig.CORS.AllowedOrigins = []string{"http://localhost:3000"}
	services.InvalidateAllCache()

	process := models.Process{Name: "Process", Code: "PROC-SEC", Type: "upper"}
	if err := database.DB.Create(&process).Error; err != nil {
		t.Fatalf("create process: %v", err)
	}
	line := models.ProductionLine{Name: "Line A", Code: "LINE-A", Type: "upper", Status: "active", ProcessID: &process.ID}
	if err := database.DB.Create(&line).Error; err != nil {
		t.Fatalf("create line: %v", err)
	}
	otherLine := models.ProductionLine{Name: "Line B", Code: "LINE-B", Type: "upper", Status: "active", ProcessID: &process.ID}
	if err := database.DB.Create(&otherLine).Error; err != nil {
		t.Fatalf("create other line: %v", err)
	}

	normal := createLineAdminSecurityUser(t, "normal", "user", nil)
	lineAdmin := createLineAdminSecurityUser(t, "line-admin", "line_admin", nil)
	admin := createLineAdminSecurityUser(t, "admin", "admin", nil)

	permRole := models.Role{Name: "permission_manager", Status: "active"}
	if err := database.DB.Create(&permRole).Error; err != nil {
		t.Fatalf("create permission role: %v", err)
	}
	perm := models.Permission{Code: "page:permissions", Name: "Permissions", Type: "page", Resource: "permission"}
	if err := database.DB.Create(&perm).Error; err != nil {
		t.Fatalf("create permission definition: %v", err)
	}
	if err := database.DB.Create(&models.RolePermission{RoleID: permRole.ID, PermissionID: perm.ID}).Error; err != nil {
		t.Fatalf("create role permission: %v", err)
	}
	permUser := createLineAdminSecurityUser(t, "perm-user", "permission_manager", &permRole.ID)

	targetUser := createLineAdminSecurityUser(t, "target", "user", nil)
	if err := database.DB.Create(&models.LineAdminAssignment{UserID: lineAdmin.ID, ProductionLineID: line.ID}).Error; err != nil {
		t.Fatalf("create line admin assignment: %v", err)
	}

	return lineAdminSecurityFixture{
		Router:      setupLineAdminSecurityRouter(),
		Line:        line,
		OtherLine:   otherLine,
		TargetUser:  targetUser,
		NormalToken: signLineAdminSecurityToken(t, normal.ID, normal.Role),
		LineToken:   signLineAdminSecurityToken(t, lineAdmin.ID, lineAdmin.Role),
		AdminToken:  signLineAdminSecurityToken(t, admin.ID, admin.Role),
		PermToken:   signLineAdminSecurityToken(t, permUser.ID, permUser.Role),
	}
}

func createLineAdminSecurityUser(t *testing.T, employeeID string, role string, roleID *uint) models.User {
	t.Helper()
	user := models.User{
		Name:       employeeID,
		Password:   "hashed",
		EmployeeID: employeeID,
		Role:       role,
		RoleID:     roleID,
		Status:     "active",
	}
	if err := database.DB.Create(&user).Error; err != nil {
		t.Fatalf("create user %s: %v", employeeID, err)
	}
	return user
}

func signLineAdminSecurityToken(t *testing.T, userID uint, role string) string {
	t.Helper()
	claims := &middleware.Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.AppConfig.Auth.JWTSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return tokenString
}

func setupLineAdminSecurityRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		lineAdmin := api.Group("/line-admin")
		{
			lineAdmin.GET("/assignments", GetLineAdminAssignments)
			lineAdmin.GET("/lines/:id/permissions", GetLinePermissionsByLine)
			lineAdmin.PUT("/lines/:id/permissions", SaveLinePermissionByAdmin)
		}
		roles := api.Group("/roles")
		{
			roles.GET("", middleware.RequirePermission("page:permissions"), GetRoles)
		}
		permissionDefs := api.Group("/permission-definitions")
		{
			permissionDefs.GET("", middleware.RequirePermission("page:permissions"), GetAllPermissions)
		}
	}
	return r
}

func performLineAdminSecurityRequest(t *testing.T, r http.Handler, method string, path string, token string, body any) *httptest.ResponseRecorder {
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
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func TestLineAdminPermissionsRejectNormalUser(t *testing.T) {
	f := setupLineAdminSecurityFixture(t)
	path := "/api/line-admin/lines/1/permissions"

	getResp := performLineAdminSecurityRequest(t, f.Router, http.MethodGet, path, f.NormalToken, nil)
	if getResp.Code != http.StatusForbidden {
		t.Fatalf("expected normal user GET status 403, got %d body=%s", getResp.Code, getResp.Body.String())
	}

	putResp := performLineAdminSecurityRequest(t, f.Router, http.MethodPut, path, f.NormalToken, gin.H{
		"user_id":      f.TargetUser.ID,
		"can_view":     true,
		"can_download": true,
		"can_upload":   true,
		"can_manage":   true,
	})
	if putResp.Code != http.StatusForbidden {
		t.Fatalf("expected normal user PUT status 403, got %d body=%s", putResp.Code, putResp.Body.String())
	}

	var userPermCount int64
	if err := database.DB.Model(&models.UserPermission{}).Where("user_id = ?", f.TargetUser.ID).Count(&userPermCount).Error; err != nil {
		t.Fatalf("count user permissions: %v", err)
	}
	if userPermCount != 0 {
		t.Fatalf("expected no user permissions to be created, got %d", userPermCount)
	}
	var ruleCount int64
	if err := database.DB.Model(&models.PermissionRule{}).Where("subject_type = ? AND subject_id = ?", models.PermissionSubjectUser, f.TargetUser.ID).Count(&ruleCount).Error; err != nil {
		t.Fatalf("count permission rules: %v", err)
	}
	if ruleCount != 0 {
		t.Fatalf("expected no permission rules to be created, got %d", ruleCount)
	}
}

func TestLineAdminCanManageOwnLineButCannotGrantManage(t *testing.T) {
	f := setupLineAdminSecurityFixture(t)
	path := "/api/line-admin/lines/1/permissions"

	getResp := performLineAdminSecurityRequest(t, f.Router, http.MethodGet, path, f.LineToken, nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("expected line admin GET status 200, got %d body=%s", getResp.Code, getResp.Body.String())
	}

	forbiddenResp := performLineAdminSecurityRequest(t, f.Router, http.MethodPut, path, f.LineToken, gin.H{
		"user_id":    f.TargetUser.ID,
		"can_view":   true,
		"can_manage": true,
	})
	if forbiddenResp.Code != http.StatusForbidden {
		t.Fatalf("expected line admin manage grant status 403, got %d body=%s", forbiddenResp.Code, forbiddenResp.Body.String())
	}

	okResp := performLineAdminSecurityRequest(t, f.Router, http.MethodPut, path, f.LineToken, gin.H{
		"user_id":      f.TargetUser.ID,
		"can_view":     true,
		"can_download": true,
		"can_upload":   true,
	})
	if okResp.Code != http.StatusOK {
		t.Fatalf("expected line admin PUT status 200, got %d body=%s", okResp.Code, okResp.Body.String())
	}

	otherLineResp := performLineAdminSecurityRequest(t, f.Router, http.MethodGet, "/api/line-admin/lines/2/permissions", f.LineToken, nil)
	if otherLineResp.Code != http.StatusForbidden {
		t.Fatalf("expected other line GET status 403, got %d body=%s", otherLineResp.Code, otherLineResp.Body.String())
	}
}

func TestAdminCanGrantManageAndRulesAreSynced(t *testing.T) {
	f := setupLineAdminSecurityFixture(t)
	resp := performLineAdminSecurityRequest(t, f.Router, http.MethodPut, "/api/line-admin/lines/1/permissions", f.AdminToken, gin.H{
		"user_id":      f.TargetUser.ID,
		"can_view":     true,
		"can_download": true,
		"can_upload":   true,
		"can_manage":   true,
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected admin PUT status 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var permission models.UserPermission
	if err := database.DB.Where("user_id = ? AND production_line_id = ?", f.TargetUser.ID, f.Line.ID).First(&permission).Error; err != nil {
		t.Fatalf("load user permission: %v", err)
	}
	if !permission.CanManage {
		t.Fatalf("expected admin to grant manage permission, got %#v", permission)
	}

	var allowManageRule models.PermissionRule
	if err := database.DB.Where(
		"subject_type = ? AND subject_id = ? AND resource_id = ? AND action = ?",
		models.PermissionSubjectUser,
		f.TargetUser.ID,
		f.Line.ID,
		models.PermissionActionManage,
	).First(&allowManageRule).Error; err != nil {
		t.Fatalf("load synced manage rule: %v", err)
	}
	if allowManageRule.Decision != models.PermissionDecisionAllow {
		t.Fatalf("expected allow manage rule, got %#v", allowManageRule)
	}
}

func TestPermissionMetadataRequiresPermissionPage(t *testing.T) {
	f := setupLineAdminSecurityFixture(t)

	for _, path := range []string{"/api/roles", "/api/permission-definitions"} {
		normalResp := performLineAdminSecurityRequest(t, f.Router, http.MethodGet, path, f.NormalToken, nil)
		if normalResp.Code != http.StatusForbidden {
			t.Fatalf("expected normal user %s status 403, got %d body=%s", path, normalResp.Code, normalResp.Body.String())
		}

		permResp := performLineAdminSecurityRequest(t, f.Router, http.MethodGet, path, f.PermToken, nil)
		if permResp.Code != http.StatusOK {
			t.Fatalf("expected permission user %s status 200, got %d body=%s", path, permResp.Code, permResp.Body.String())
		}
	}
}
