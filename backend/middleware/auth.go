package middleware

import (
	"crane-system/config"
	"crane-system/database"
	"crane-system/models"
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

// AuthMiddleware 校验 Bearer JWT，并把 user_id/user_role/user 写入 Gin Context。
// 每次请求都会重新读取用户状态，确保禁用账号或角色变更能及时生效。
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证令牌"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "令牌格式错误"})
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

// AdminMiddleware 只依赖 AuthMiddleware 写入的 user_role。
// 使用时应放在受保护路由之后，不能单独挂在公共路由上。
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("user_role")
		if !exists || role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
			c.Abort()
			return
		}
		c.Next()
	}
}
