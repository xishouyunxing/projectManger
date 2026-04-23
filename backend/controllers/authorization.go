package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type linePermissionAction string

const (
	lineActionView     linePermissionAction = "view"
	lineActionDownload linePermissionAction = "download"
	lineActionUpload   linePermissionAction = "upload"
	lineActionManage   linePermissionAction = "manage"
)

func authorizeOwnerOrAdmin(c *gin.Context, targetUserID uint) bool {
	roleValue, roleExists := c.Get("user_role")
	if roleExists {
		if role, ok := roleValue.(string); ok && role == "admin" {
			return true
		}
	}

	userIDValue, userIDExists := c.Get("user_id")
	if !userIDExists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证用户"})
		return false
	}
	userID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户身份无效"})
		return false
	}
	if userID != targetUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权限"})
		return false
	}
	return true
}

func authorizeProgramAction(c *gin.Context, tx *gorm.DB, programID uint, action linePermissionAction) bool {
	var program models.Program
	if err := tx.Select("id", "production_line_id").First(&program, programID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
			return false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return false
	}
	return authorizeLineAction(c, program.ProductionLineID, action)
}

func checkLineAction(c *gin.Context, productionLineID uint, action linePermissionAction) (bool, int, string) {
	roleValue, roleExists := c.Get("user_role")
	if roleExists {
		if role, ok := roleValue.(string); ok && role == "admin" {
			return true, 0, ""
		}
	}

	userIDValue, userIDExists := c.Get("user_id")
	if !userIDExists {
		return false, http.StatusUnauthorized, "未认证用户"
	}
	userID, ok := userIDValue.(uint)
	if !ok {
		return false, http.StatusUnauthorized, "用户身份无效"
	}

	var user models.User
	if err := database.DB.Select("id", "department_id").First(&user, userID).Error; err != nil {
		return false, http.StatusUnauthorized, "用户不存在"
	}

	canView := false
	canDownload := false
	canUpload := false
	canManage := false

	var userPermission models.UserPermission
	err := database.DB.Where("user_id = ? AND production_line_id = ?", user.ID, productionLineID).First(&userPermission).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return false, http.StatusInternalServerError, "查询权限失败"
	}
	if err == nil {
		canView = canView || userPermission.CanView
		canDownload = canDownload || userPermission.CanDownload
		canUpload = canUpload || userPermission.CanUpload
		canManage = canManage || userPermission.CanManage
	}

	if user.DepartmentID != nil {
		var departmentPermission models.DepartmentPermission
		err = database.DB.Where("department_id = ? AND production_line_id = ?", *user.DepartmentID, productionLineID).First(&departmentPermission).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return false, http.StatusInternalServerError, "查询权限失败"
		}
		if err == nil {
			canView = canView || departmentPermission.CanView
			canDownload = canDownload || departmentPermission.CanDownload
			canUpload = canUpload || departmentPermission.CanUpload
			canManage = canManage || departmentPermission.CanManage
		}
	}

	allowed := false
	switch action {
	case lineActionView:
		allowed = canView || canManage
	case lineActionDownload:
		allowed = canDownload || canManage
	case lineActionUpload:
		allowed = canUpload || canManage
	case lineActionManage:
		allowed = canManage
	default:
		allowed = false
	}

	if !allowed {
		return false, http.StatusForbidden, "无权限"
	}

	return true, 0, ""
}

func authorizeLineAction(c *gin.Context, productionLineID uint, action linePermissionAction) bool {
	allowed, statusCode, message := checkLineAction(c, productionLineID, action)
	if !allowed {
		c.JSON(statusCode, gin.H{"error": message})
		return false
	}
	return true
}
