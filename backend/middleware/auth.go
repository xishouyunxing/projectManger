package middleware

import (
	"crane-system/config"
	"crane-system/database"
	"crane-system/models"
	"crane-system/services"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID uint   `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

const authTokenCookieName = "auth_token"

func tokenFromRequest(c *gin.Context) (string, string) {
	if cookie, err := c.Cookie(authTokenCookieName); err == nil && cookie != "" {
		return cookie, ""
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", "未提供认证令牌"
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		return "", "令牌格式错误"
	}

	return tokenString, ""
}

// AuthMiddleware validates either the HttpOnly auth cookie or a legacy Bearer JWT.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, tokenErr := tokenFromRequest(c)
		if tokenErr != "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": tokenErr})
			c.Abort()
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.AppConfig.Auth.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的令牌"})
			c.Abort()
			return
		}

		var user models.User
		if err := database.DB.First(&user, claims.UserID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
			c.Abort()
			return
		}

		if user.Status != "active" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户已被禁用"})
			c.Abort()
			return
		}

		if claims.Role != "" && claims.Role != user.Role {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "令牌角色与当前用户角色不一致"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_role", user.Role)
		c.Set("user", user)
		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists || (role != "admin" && role != "system_admin") {
			c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequirePermission(permissionCode string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("user_role")
		if role == "admin" || role == "system_admin" {
			c.Next()
			return
		}

		userIDVal, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
			c.Abort()
			return
		}
		userID, ok := userIDVal.(uint)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户ID无效"})
			c.Abort()
			return
		}

		if !services.UserHasPermission(userID, permissionCode) {
			c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequireAnyPermission(permissionCodes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("user_role")
		if role == "admin" || role == "system_admin" {
			c.Next()
			return
		}

		userIDVal, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
			c.Abort()
			return
		}
		userID, ok := userIDVal.(uint)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户ID无效"})
			c.Abort()
			return
		}

		for _, code := range permissionCodes {
			if services.UserHasPermission(userID, code) {
				c.Next()
				return
			}
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		c.Abort()
	}
}
