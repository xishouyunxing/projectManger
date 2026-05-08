package controllers

import (
	"crane-system/models"
	"crane-system/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type linePermissionAction = services.LineAction

const (
	lineActionView     linePermissionAction = services.LineActionView
	lineActionDownload linePermissionAction = services.LineActionDownload
	lineActionUpload   linePermissionAction = services.LineActionUpload
	lineActionManage   linePermissionAction = services.LineActionManage
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
	return writeAuthDecision(c, services.AuthorizeOwnerOrAdmin(currentUserID(c), currentUserRole(c), targetUserID))
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
	decision := services.CheckLineAction(currentUserID(c), currentUserRole(c), productionLineID, action)
	return decision.Allowed, decision.StatusCode, decision.Message
}

// resolveAuthorizedLineIDs 返回当前用户可访问的产线集合。
// 管理员返回 nil 表示不需要追加产线过滤条件。
func resolveAuthorizedLineIDs(c *gin.Context, action linePermissionAction) (map[uint]struct{}, int, string) {
	allowedLineIDs, decision := services.ResolveAuthorizedLineIDs(currentUserID(c), currentUserRole(c), action)
	if !decision.Allowed {
		return nil, decision.StatusCode, decision.Message
	}
	return allowedLineIDs, 0, ""
}

func permissionAllowsAction(canView, canDownload, canUpload, canManage bool, action linePermissionAction) bool {
	return services.LinePermissionAllowsAction(canView, canDownload, canUpload, canManage, action)
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

func currentUserRole(c *gin.Context) string {
	roleValue, _ := c.Get("user_role")
	role, _ := roleValue.(string)
	return role
}

func currentUserID(c *gin.Context) uint {
	return c.GetUint("user_id")
}

func writeAuthDecision(c *gin.Context, decision services.AuthDecision) bool {
	if decision.Allowed {
		return true
	}
	c.JSON(decision.StatusCode, gin.H{"error": decision.Message})
	return false
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
