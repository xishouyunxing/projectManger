package controllers

import (
	"bytes"
	"crane-system/config"
	"crane-system/database"
	"crane-system/middleware"
	"crane-system/models"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func seedAuthTestUser(t *testing.T) models.User {
	t.Helper()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	user := models.User{
		EmployeeID: "AUTH-1",
		Name:       "Auth User",
		Role:       "admin",
		Password:   string(hashedPassword),
		Status:     "active",
	}
	if err := database.DB.Create(&user).Error; err != nil {
		t.Fatalf("create auth user: %v", err)
	}
	return user
}

func setupAuthTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api")
	api.POST("/login", Login)
	api.POST("/logout", Logout)
	api.GET("/protected", middleware.AuthMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func performJSONRequest(r *gin.Engine, method, path string, payload any) *httptest.ResponseRecorder {
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	return resp
}

func signAuthTestToken(t *testing.T, user models.User) string {
	t.Helper()

	claims := &middleware.Claims{
		UserID: user.ID,
		Role:   user.Role,
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

func TestLoginSetsHttpOnlyAuthCookie(t *testing.T) {
	database.DB = openProductionLineCustomFieldTestDB(t)
	config.AppConfig.Auth.JWTSecret = "auth-test-secret"
	seedAuthTestUser(t)
	r := setupAuthTestRouter()

	resp := performJSONRequest(r, "POST", "/api/login", gin.H{
		"employee_id": "AUTH-1",
		"password":    "secret",
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var authCookie *http.Cookie
	for _, cookie := range resp.Result().Cookies() {
		if cookie.Name == authTokenCookieName {
			authCookie = cookie
			break
		}
	}
	if authCookie == nil {
		t.Fatalf("expected auth cookie")
	}
	if !authCookie.HttpOnly {
		t.Fatalf("expected auth cookie to be HttpOnly")
	}
	if authCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite=Lax, got %v", authCookie.SameSite)
	}
	if authCookie.MaxAge <= 0 {
		t.Fatalf("expected positive MaxAge, got %d", authCookie.MaxAge)
	}
}

func TestAuthMiddlewareAcceptsCookieToken(t *testing.T) {
	database.DB = openProductionLineCustomFieldTestDB(t)
	config.AppConfig.Auth.JWTSecret = "auth-test-secret"
	user := seedAuthTestUser(t)
	r := setupAuthTestRouter()

	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.AddCookie(&http.Cookie{Name: authTokenCookieName, Value: signAuthTestToken(t, user)})
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestAuthMiddlewareStillAcceptsBearerToken(t *testing.T) {
	database.DB = openProductionLineCustomFieldTestDB(t)
	config.AppConfig.Auth.JWTSecret = "auth-test-secret"
	user := seedAuthTestUser(t)
	r := setupAuthTestRouter()

	req := httptest.NewRequest("GET", "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+signAuthTestToken(t, user))
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestLogoutClearsAuthCookie(t *testing.T) {
	r := setupAuthTestRouter()

	resp := performJSONRequest(r, "POST", "/api/logout", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var authCookie *http.Cookie
	for _, cookie := range resp.Result().Cookies() {
		if cookie.Name == authTokenCookieName {
			authCookie = cookie
			break
		}
	}
	if authCookie == nil {
		t.Fatalf("expected auth cookie")
	}
	if authCookie.MaxAge >= 0 {
		t.Fatalf("expected clearing cookie MaxAge < 0, got %d", authCookie.MaxAge)
	}
}
