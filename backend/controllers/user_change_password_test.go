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
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func createUserTokenForTest(t *testing.T, userID uint, role string) string {
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
		t.Fatalf("sign jwt: %v", err)
	}
	return tokenString
}

func setupChangePasswordTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		users := api.Group("/users")
		users.POST("/:id/change-password", ChangePassword)
	}
	return r
}

func setupUpdateUserTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		users := api.Group("/users")
		users.POST("", CreateUser)
		users.PUT("/:id", UpdateUser)
	}
	return r
}

func performChangePasswordRequest(t *testing.T, r http.Handler, token string, targetUserID uint, payload any) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request payload: %v", err)
	}

	path := fmt.Sprintf("/api/users/%d/change-password", targetUserID)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func performUpdateUserRequest(t *testing.T, r http.Handler, token string, targetUserID uint, payload any) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal request payload: %v", err)
	}

	path := fmt.Sprintf("/api/users/%d", targetUserID)
	req := httptest.NewRequest(http.MethodPut, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func performCreateUserRequest(t *testing.T, r http.Handler, token string, payload any) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func createHashedPasswordForTest(t *testing.T, plain string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	return string(hash)
}

func assertPasswordMatches(t *testing.T, userID uint, plain string) {
	t.Helper()
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		t.Fatalf("load user: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(plain)); err != nil {
		t.Fatalf("password mismatch for user %d", userID)
	}
}

func TestChangePasswordAuthorizationAndBoundaries(t *testing.T) {
	database.DB = openProductionLineCustomFieldTestDB(t)
	r := setupChangePasswordTestRouter()

	admin := models.User{Name: "Admin", EmployeeID: "EMP-A", Role: "admin", Password: createHashedPasswordForTest(t, "admin-old"), Status: "active"}
	selfUser := models.User{Name: "Self", EmployeeID: "EMP-S", Role: "user", Password: createHashedPasswordForTest(t, "self-old"), Status: "active"}
	otherUser := models.User{Name: "Other", EmployeeID: "EMP-O", Role: "user", Password: createHashedPasswordForTest(t, "other-old"), Status: "active"}
	if err := database.DB.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if err := database.DB.Create(&selfUser).Error; err != nil {
		t.Fatalf("create self user: %v", err)
	}
	if err := database.DB.Create(&otherUser).Error; err != nil {
		t.Fatalf("create other user: %v", err)
	}

	selfToken := createUserTokenForTest(t, selfUser.ID, "user")
	adminToken := createUserTokenForTest(t, admin.ID, "admin")

	t.Run("rejects non-self non-admin", func(t *testing.T) {
		resp := performChangePasswordRequest(t, r, selfToken, otherUser.ID, map[string]any{
			"old_password": "self-old",
			"new_password": "new-pass-1",
		})
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d body=%s", resp.Code, resp.Body.String())
		}
		assertPasswordMatches(t, otherUser.ID, "other-old")
	})

	t.Run("allows self change with old password", func(t *testing.T) {
		resp := performChangePasswordRequest(t, r, selfToken, selfUser.ID, map[string]any{
			"old_password": "self-old",
			"new_password": "self-new-1",
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
		}
		assertPasswordMatches(t, selfUser.ID, "self-new-1")
	})

	t.Run("requires old password for self", func(t *testing.T) {
		resp := performChangePasswordRequest(t, r, selfToken, selfUser.ID, map[string]any{
			"new_password": "self-new-2",
		})
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("allows admin reset without old password", func(t *testing.T) {
		resp := performChangePasswordRequest(t, r, adminToken, otherUser.ID, map[string]any{
			"new_password": "admin-reset-1",
		})
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
		}
		assertPasswordMatches(t, otherUser.ID, "admin-reset-1")
	})
}

func TestAuthMiddlewareRejectsTokenRoleMismatch(t *testing.T) {
	database.DB = openProductionLineCustomFieldTestDB(t)
	r := setupChangePasswordTestRouter()

	admin := models.User{Name: "Admin", EmployeeID: "EMP-A2", Role: "admin", Password: createHashedPasswordForTest(t, "admin-old"), Status: "active"}
	target := models.User{Name: "Target", EmployeeID: "EMP-T2", Role: "user", Password: createHashedPasswordForTest(t, "target-old"), Status: "active"}
	if err := database.DB.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if err := database.DB.Create(&target).Error; err != nil {
		t.Fatalf("create target: %v", err)
	}

	mismatchToken := createUserTokenForTest(t, admin.ID, "user")
	resp := performChangePasswordRequest(t, r, mismatchToken, target.ID, map[string]any{
		"new_password": "never-applied",
	})
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", resp.Code, resp.Body.String())
	}
	assertPasswordMatches(t, target.ID, "target-old")
}

func TestUpdateUserDoesNotAllowSelfServiceDepartmentEscalation(t *testing.T) {
	database.DB = openProductionLineCustomFieldTestDB(t)
	r := setupUpdateUserTestRouter()

	deptA := models.Department{Name: "Dept-A", Status: "active"}
	deptB := models.Department{Name: "Dept-B", Status: "active"}
	if err := database.DB.Create(&deptA).Error; err != nil {
		t.Fatalf("create deptA: %v", err)
	}
	if err := database.DB.Create(&deptB).Error; err != nil {
		t.Fatalf("create deptB: %v", err)
	}

	selfUser := models.User{
		Name:         "Self",
		EmployeeID:   "EMP-U1",
		Role:         "user",
		Password:     createHashedPasswordForTest(t, "self-old"),
		Status:       "active",
		DepartmentID: &deptA.ID,
	}
	adminUser := models.User{
		Name:       "Admin",
		EmployeeID: "EMP-U2",
		Role:       "admin",
		Password:   createHashedPasswordForTest(t, "admin-old"),
		Status:     "active",
	}
	if err := database.DB.Create(&selfUser).Error; err != nil {
		t.Fatalf("create self user: %v", err)
	}
	if err := database.DB.Create(&adminUser).Error; err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	selfToken := createUserTokenForTest(t, selfUser.ID, "user")
	adminToken := createUserTokenForTest(t, adminUser.ID, "admin")

	resp := performUpdateUserRequest(t, r, selfToken, selfUser.ID, map[string]any{
		"name":          "Self Updated",
		"department_id": deptB.ID,
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 for self update, got %d body=%s", resp.Code, resp.Body.String())
	}

	var reloaded models.User
	if err := database.DB.First(&reloaded, selfUser.ID).Error; err != nil {
		t.Fatalf("reload self user: %v", err)
	}
	if reloaded.Name != "Self Updated" {
		t.Fatalf("expected self-service name update to persist, got %q", reloaded.Name)
	}
	if reloaded.DepartmentID == nil || *reloaded.DepartmentID != deptA.ID {
		t.Fatalf("expected self-service department to remain %d, got %#v", deptA.ID, reloaded.DepartmentID)
	}

	resp = performUpdateUserRequest(t, r, adminToken, selfUser.ID, map[string]any{
		"department_id": deptB.ID,
		"status":        "inactive",
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin update, got %d body=%s", resp.Code, resp.Body.String())
	}

	if err := database.DB.First(&reloaded, selfUser.ID).Error; err != nil {
		t.Fatalf("reload updated user: %v", err)
	}
	if reloaded.DepartmentID == nil || *reloaded.DepartmentID != deptB.ID {
		t.Fatalf("expected admin to move department to %d, got %#v", deptB.ID, reloaded.DepartmentID)
	}
	if reloaded.Status != "inactive" {
		t.Fatalf("expected admin status update to persist, got %q", reloaded.Status)
	}
}

func TestCreateUserValidatesRoleStatusAndDepartment(t *testing.T) {
	database.DB = openProductionLineCustomFieldTestDB(t)
	r := setupUpdateUserTestRouter()

	adminUser := models.User{
		Name:       "Admin",
		EmployeeID: "EMP-CREATE-ADMIN",
		Role:       "admin",
		Password:   createHashedPasswordForTest(t, "admin-old"),
		Status:     "active",
	}
	if err := database.DB.Create(&adminUser).Error; err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	adminToken := createUserTokenForTest(t, adminUser.ID, "admin")

	department := models.Department{Name: "Valid Dept", Status: "active"}
	if err := database.DB.Create(&department).Error; err != nil {
		t.Fatalf("create department: %v", err)
	}

	basePayload := map[string]any{
		"employee_id":   "EMP-NEW",
		"name":          "New User",
		"password":      "new-password",
		"role":          "user",
		"status":        "active",
		"department_id": department.ID,
	}

	resp := performCreateUserRequest(t, r, adminToken, basePayload)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d body=%s", resp.Code, resp.Body.String())
	}

	invalidDepartmentPayload := map[string]any{}
	for key, value := range basePayload {
		invalidDepartmentPayload[key] = value
	}
	invalidDepartmentPayload["employee_id"] = "EMP-BAD-DEPT"
	invalidDepartmentPayload["department_id"] = department.ID + 999
	resp = performCreateUserRequest(t, r, adminToken, invalidDepartmentPayload)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid department status 400, got %d body=%s", resp.Code, resp.Body.String())
	}

	invalidRolePayload := map[string]any{}
	for key, value := range basePayload {
		invalidRolePayload[key] = value
	}
	invalidRolePayload["employee_id"] = "EMP-BAD-ROLE"
	invalidRolePayload["role"] = "superuser"
	resp = performCreateUserRequest(t, r, adminToken, invalidRolePayload)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid role status 400, got %d body=%s", resp.Code, resp.Body.String())
	}

	invalidStatusPayload := map[string]any{}
	for key, value := range basePayload {
		invalidStatusPayload[key] = value
	}
	invalidStatusPayload["employee_id"] = "EMP-BAD-STATUS"
	invalidStatusPayload["status"] = "pending"
	resp = performCreateUserRequest(t, r, adminToken, invalidStatusPayload)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid status status 400, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestUpdateUserRejectsInvalidDepartment(t *testing.T) {
	database.DB = openProductionLineCustomFieldTestDB(t)
	r := setupUpdateUserTestRouter()

	adminUser := models.User{Name: "Admin", EmployeeID: "EMP-DEPT-ADMIN", Role: "admin", Password: createHashedPasswordForTest(t, "admin-old"), Status: "active"}
	targetUser := models.User{Name: "Target", EmployeeID: "EMP-DEPT-TARGET", Role: "user", Password: createHashedPasswordForTest(t, "target-old"), Status: "active"}
	if err := database.DB.Create(&adminUser).Error; err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	if err := database.DB.Create(&targetUser).Error; err != nil {
		t.Fatalf("create target user: %v", err)
	}

	adminToken := createUserTokenForTest(t, adminUser.ID, "admin")
	resp := performUpdateUserRequest(t, r, adminToken, targetUser.ID, map[string]any{
		"department_id": uint(999),
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid department update status 400, got %d body=%s", resp.Code, resp.Body.String())
	}
}
