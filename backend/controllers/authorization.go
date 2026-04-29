package controllers

import (
	"crane-system/models"
	"crane-system/services"
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

type resolvedLinePermission struct {
	ProductionLineID uint
	CanView          bool
	CanDownload      bool
	CanUpload        bool
	CanManage        bool
	Source           string
}

// authorizeOwnerOrAdmin 用于"本人可访问、管理员可访问"的账号级接口。
func authorizeOwnerOrAdmin(c *gin.Context, targetUserID uint) bool {
	roleValue, roleExists := c.Get("user_role")
	if roleExists {
		if role, ok := roleValue.(string); ok && (role == "admin" || role == "system_admin") {
			return true
		}
	}

	userIDValue, userIDExists := c.Get("user_id")
	if !userIDExists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return false
	}
	userID, ok := userIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户ID无效"})
		return false
	}
	if userID != targetUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作"})
		return false
	}
	return true
}

// authorizeProgramAction 将程序级操作转换为产线权限判断。
// 程序本身不直接存权限，所有业务授权都回到 ProductionLineID。
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

// checkLineAction 返回可直接用于接口响应的授权结果。
// system_admin 硬编码绕过；line_admin 对负责产线全权；普通用户按用户覆盖→角色权限解析。
func checkLineAction(c *gin.Context, productionLineID uint, action linePermissionAction) (bool, int, string) {
	roleValue, _ := c.Get("user_role")
	role, _ := roleValue.(string)

	// system_admin 硬编码绕过
	if role == "admin" || role == "system_admin" {
		return true, 0, ""
	}

	userIDValue, userIDExists := c.Get("user_id")
	if !userIDExists {
		return false, http.StatusUnauthorized, "未认证"
	}
	userID, ok := userIDValue.(uint)
	if !ok {
		return false, http.StatusUnauthorized, "用户ID无效"
	}

	// line_admin 对负责的产线有管理权
	if role == "line_admin" && services.IsLineManager(userID, productionLineID) {
		return true, 0, ""
	}

	// 使用缓存权限服务解析：用户覆盖 → 角色产线权限
	allowed := services.UserHasLinePermission(userID, productionLineID, string(action))
	if !allowed {
		return false, http.StatusForbidden, "无权操作"
	}

	return true, 0, ""
}

// resolveAuthorizedLineIDs 返回当前用户可访问的产线集合。
// 管理员返回 nil 表示不需要追加产线过滤条件。
func resolveAuthorizedLineIDs(c *gin.Context, action linePermissionAction) (map[uint]struct{}, int, string) {
	roleValue, _ := c.Get("user_role")
	role, _ := roleValue.(string)

	// admin 不需要过滤
	if role == "admin" || role == "system_admin" {
		return nil, 0, ""
	}

	userIDValue, userIDExists := c.Get("user_id")
	if !userIDExists {
		return nil, http.StatusUnauthorized, "未认证"
	}
	userID, ok := userIDValue.(uint)
	if !ok {
		return nil, http.StatusUnauthorized, "用户ID无效"
	}

	perms, err := services.GetUserPermissions(userID)
	if err != nil {
		return nil, http.StatusInternalServerError, "查询权限失败"
	}

	allowedLineIDs := map[uint]struct{}{}
	for lineID, lp := range perms.LinePermissions {
		if permissionAllowsAction(lp.CanView, lp.CanDownload, lp.CanUpload, lp.CanManage, action) {
			allowedLineIDs[lineID] = struct{}{}
		}
	}

	// line_admin 对负责的产线也有权限
	if role == "line_admin" {
		for _, managedID := range perms.ManagedLineIDs {
			allowedLineIDs[managedID] = struct{}{}
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

// resolveUserLinePermission 供测试使用的单产线权限解析。
func resolveUserLinePermission(user models.User, productionLineID uint) (resolvedLinePermission, error) {
	perms, err := services.GetUserPermissions(user.ID)
	if err != nil {
		return resolvedLinePermission{ProductionLineID: productionLineID, Source: "none"}, err
	}
	lp, ok := perms.LinePermissions[productionLineID]
	if !ok {
		return resolvedLinePermission{ProductionLineID: productionLineID, Source: "none"}, nil
	}
	return resolvedLinePermission{
		ProductionLineID: productionLineID,
		CanView:          lp.CanView,
		CanDownload:      lp.CanDownload,
		CanUpload:        lp.CanUpload,
		CanManage:        lp.CanManage,
		Source:           lp.Source,
	}, nil
}
