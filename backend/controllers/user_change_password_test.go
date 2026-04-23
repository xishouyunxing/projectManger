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
