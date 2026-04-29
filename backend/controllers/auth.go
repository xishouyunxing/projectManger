package controllers

import (
	"crane-system/config"
	"crane-system/database"
	"crane-system/middleware"
	"crane-system/models"
	"crane-system/services"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	EmployeeID string `json:"employee_id" binding:"required"`
	Password   string `json:"password" binding:"required"`
}

// Login 校验工号和密码并签发 JWT。
// 返回的 user 会被前端缓存到 AuthContext，后续请求通过 Authorization Bearer token 鉴权。
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := database.DB.Preload("Department").Where("employee_id = ?", req.EmployeeID).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "工号或密码错误"})
		return
	}

	if user.Status != "active" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户已被禁用"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "工号或密码错误"})
		return
	}

	claims := &middleware.Claims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.AppConfig.Auth.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	// 加载用户权限数据
	permData, err := services.GetUserPermissions(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载权限失败"})
		return
	}

	// 构建产线权限响应
	linePerms := make(map[string]gin.H)
	lineIDs := make([]string, 0)
	for lineID, lp := range permData.LinePermissions {
		key := strconv.FormatUint(uint64(lineID), 10)
		linePerms[key] = gin.H{
			"can_view":     lp.CanView,
			"can_download": lp.CanDownload,
			"can_upload":   lp.CanUpload,
			"can_manage":   lp.CanManage,
		}
		lineIDs = append(lineIDs, key)
	}

	// 构建 managed_line_ids
	managedIDs := make([]string, 0, len(permData.ManagedLineIDs))
	for _, id := range permData.ManagedLineIDs {
		managedIDs = append(managedIDs, strconv.FormatUint(uint64(id), 10))
	}

	// 获取角色ID
	var roleIDPtr *uint
	if user.RoleID != nil {
		roleIDPtr = user.RoleID
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
		"user": gin.H{
			"id":          user.ID,
			"employee_id": user.EmployeeID,
			"name":        user.Name,
			"department":  user.Department,
			"role":        user.Role,
			"role_id":     roleIDPtr,
		},
		"permissions": gin.H{
			"codes":            permData.FunctionCodes,
			"lines":            linePerms,
			"managed_line_ids": managedIDs,
		},
	})
}
