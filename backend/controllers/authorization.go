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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "?????"})
		return false
	}
	userID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "??????"})
		return false
	}
	if userID != targetUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "???"})
		return false
	}
	return true
}

func authorizeProgramAction(c *gin.Context, tx *gorm.DB, programID uint, action linePermissionAction) bool {
	var program models.Program
	if err := tx.Select("id", "production_line_id").First(&program, programID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "?????"})
			return false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
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
		return false, http.StatusUnauthorized, "?????"
	}
	userID, ok := userIDValue.(uint)
	if !ok {
		return false, http.StatusUnauthorized, "??????"
	}

	var user models.User
	if err := database.DB.Select("id", "department_id").First(&user, userID).Error; err != nil {
		return false, http.StatusUnauthorized, "?????"
	}

	canView := false
	canDownload := false
	canUpload := false
	canManage := false

	var userPermissions []models.UserPermission
	err := database.DB.Where("user_id = ? AND production_line_id = ?", user.ID, productionLineID).Find(&userPermissions).Error
	if err != nil {
		return false, http.StatusInternalServerError, "??????"
	}
	for _, userPermission := range userPermissions {
		canView = canView || userPermission.CanView
		canDownload = canDownload || userPermission.CanDownload
		canUpload = canUpload || userPermission.CanUpload
		canManage = canManage || userPermission.CanManage
	}

	if user.DepartmentID != nil {
		var departmentPermission models.DepartmentPermission
		err = database.DB.Where("department_id = ? AND production_line_id = ?", *user.DepartmentID, productionLineID).First(&departmentPermission).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return false, http.StatusInternalServerError, "??????"
		}
		if err == nil {
			canView = canView || departmentPermission.CanView
			canDownload = canDownload || departmentPermission.CanDownload
			canUpload = canUpload || departmentPermission.CanUpload
			canManage = canManage || departmentPermission.CanManage
		}
	}

	allowed := permissionAllowsAction(canView, canDownload, canUpload, canManage, action)
	if !allowed {
		return false, http.StatusForbidden, "???"
	}

	return true, 0, ""
}

func resolveAuthorizedLineIDs(c *gin.Context, action linePermissionAction) (map[uint]struct{}, int, string) {
	roleValue, roleExists := c.Get("user_role")
	if roleExists {
		if role, ok := roleValue.(string); ok && role == "admin" {
			return nil, 0, ""
		}
	}

	userIDValue, userIDExists := c.Get("user_id")
	if !userIDExists {
		return nil, http.StatusUnauthorized, "?????"
	}
	userID, ok := userIDValue.(uint)
	if !ok {
		return nil, http.StatusUnauthorized, "??????"
	}

	var user models.User
	if err := database.DB.Select("id", "department_id").First(&user, userID).Error; err != nil {
		return nil, http.StatusUnauthorized, "?????"
	}

	allowedLineIDs := map[uint]struct{}{}

	var userPermissions []models.UserPermission
	if err := database.DB.
		Select("production_line_id", "can_view", "can_download", "can_upload", "can_manage").
		Where("user_id = ?", user.ID).
		Find(&userPermissions).Error; err != nil {
		return nil, http.StatusInternalServerError, "??????"
	}
	for _, permission := range userPermissions {
		if permissionAllowsAction(permission.CanView, permission.CanDownload, permission.CanUpload, permission.CanManage, action) {
			allowedLineIDs[permission.ProductionLineID] = struct{}{}
		}
	}

	if user.DepartmentID != nil {
		var departmentPermissions []models.DepartmentPermission
		if err := database.DB.
			Select("production_line_id", "can_view", "can_download", "can_upload", "can_manage").
			Where("department_id = ?", *user.DepartmentID).
			Find(&departmentPermissions).Error; err != nil {
			return nil, http.StatusInternalServerError, "??????"
		}
		for _, permission := range departmentPermissions {
			if permissionAllowsAction(permission.CanView, permission.CanDownload, permission.CanUpload, permission.CanManage, action) {
				allowedLineIDs[permission.ProductionLineID] = struct{}{}
			}
		}
	}

	return allowedLineIDs, 0, ""
}

func permissionAllowsAction(canView, canDownload, canUpload, canManage bool, action linePermissionAction) bool {
	switch action {
	case lineActionView:
		return canView || canManage
	case lineActionDownload:
		return canDownload || canManage
	case lineActionUpload:
		return canUpload || canManage
	case lineActionManage:
		return canManage
	default:
		return false
	}
}

func lineIDAllowed(allowedLineIDs map[uint]struct{}, lineID uint) bool {
	if allowedLineIDs == nil {
		return true
	}
	_, ok := allowedLineIDs[lineID]
	return ok
}

func authorizeLineAction(c *gin.Context, productionLineID uint, action linePermissionAction) bool {
	allowed, statusCode, message := checkLineAction(c, productionLineID, action)
	if !allowed {
		c.JSON(statusCode, gin.H{"error": message})
		return false
	}
	return true
}
