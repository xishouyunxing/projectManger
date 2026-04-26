package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"net/http"
	"strings"

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
	if err := database.DB.Select("id", "department_id", "role").First(&user, userID).Error; err != nil {
		return false, http.StatusUnauthorized, "?????"
	}

	permission, err := resolveUserLinePermission(user, productionLineID)
	if err != nil {
		return false, http.StatusInternalServerError, "??????"
	}

	allowed := permissionAllowsAction(permission.CanView, permission.CanDownload, permission.CanUpload, permission.CanManage, action)
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
	if err := database.DB.Select("id", "department_id", "role").First(&user, userID).Error; err != nil {
		return nil, http.StatusUnauthorized, "?????"
	}

	var productionLines []models.ProductionLine
	if err := database.DB.Select("id").Find(&productionLines).Error; err != nil {
		return nil, http.StatusInternalServerError, "??????"
	}
	permissions, err := resolveUserLinePermissions(user, productionLines)
	if err != nil {
		return nil, http.StatusInternalServerError, "??????"
	}

	allowedLineIDs := map[uint]struct{}{}
	for _, permission := range permissions {
		if permissionAllowsAction(permission.CanView, permission.CanDownload, permission.CanUpload, permission.CanManage, action) {
			allowedLineIDs[permission.ProductionLineID] = struct{}{}
		}
	}

	return allowedLineIDs, 0, ""
}

func resolveUserLinePermission(user models.User, productionLineID uint) (resolvedLinePermission, error) {
	permissions, err := resolveUserLinePermissions(user, []models.ProductionLine{{ID: productionLineID}})
	if err != nil {
		return resolvedLinePermission{}, err
	}
	if len(permissions) == 0 {
		return resolvedLinePermission{ProductionLineID: productionLineID, Source: "none"}, nil
	}
	return permissions[0], nil
}

func resolveUserLinePermissions(user models.User, productionLines []models.ProductionLine) ([]resolvedLinePermission, error) {
	lineIDs := make([]uint, 0, len(productionLines))
	for _, line := range productionLines {
		lineIDs = append(lineIDs, line.ID)
	}
	if len(lineIDs) == 0 {
		return []resolvedLinePermission{}, nil
	}

	userPerms := []models.UserPermission{}
	if err := database.DB.
		Where("user_id = ? AND production_line_id IN ?", user.ID, lineIDs).
		Find(&userPerms).Error; err != nil {
		return nil, err
	}
	userPermMap := make(map[uint]models.UserPermission, len(userPerms))
	for _, perm := range userPerms {
		userPermMap[perm.ProductionLineID] = perm
	}

	deptPermMap := map[uint]models.DepartmentPermission{}
	if user.DepartmentID != nil {
		deptPerms := []models.DepartmentPermission{}
		if err := database.DB.
			Where("department_id = ? AND production_line_id IN ?", *user.DepartmentID, lineIDs).
			Find(&deptPerms).Error; err != nil {
			return nil, err
		}
		for _, perm := range deptPerms {
			deptPermMap[perm.ProductionLineID] = perm
		}
	}

	roleDefaultMap := map[uint]models.RoleDefaultPermission{}
	if role := strings.TrimSpace(user.Role); role != "" {
		roleDefaults := []models.RoleDefaultPermission{}
		if err := database.DB.
			Where("role = ? AND production_line_id IN ?", role, lineIDs).
			Find(&roleDefaults).Error; err != nil {
			return nil, err
		}
		for _, perm := range roleDefaults {
			roleDefaultMap[perm.ProductionLineID] = perm
		}
	}

	deptDefaultMap := map[uint]models.DepartmentDefaultPermission{}
	if user.DepartmentID != nil {
		deptDefaults := []models.DepartmentDefaultPermission{}
		if err := database.DB.
			Where("department_id = ? AND production_line_id IN ?", *user.DepartmentID, lineIDs).
			Find(&deptDefaults).Error; err != nil {
			return nil, err
		}
		for _, perm := range deptDefaults {
			deptDefaultMap[perm.ProductionLineID] = perm
		}
	}

	resolved := make([]resolvedLinePermission, 0, len(productionLines))
	for _, line := range productionLines {
		permission := resolvedLinePermission{ProductionLineID: line.ID, Source: "none"}
		if perm, ok := userPermMap[line.ID]; ok {
			permission.CanView = perm.CanView
			permission.CanDownload = perm.CanDownload
			permission.CanUpload = perm.CanUpload
			permission.CanManage = perm.CanManage
			permission.Source = "user"
		} else if perm, ok := deptPermMap[line.ID]; ok {
			permission.CanView = perm.CanView
			permission.CanDownload = perm.CanDownload
			permission.CanUpload = perm.CanUpload
			permission.CanManage = perm.CanManage
			permission.Source = "department"
		} else if perm, ok := roleDefaultMap[line.ID]; ok {
			permission.CanView = perm.CanView
			permission.CanDownload = perm.CanDownload
			permission.CanUpload = perm.CanUpload
			permission.CanManage = perm.CanManage
			permission.Source = "role_default"
		} else if perm, ok := deptDefaultMap[line.ID]; ok {
			permission.CanView = perm.CanView
			permission.CanDownload = perm.CanDownload
			permission.CanUpload = perm.CanUpload
			permission.CanManage = perm.CanManage
			permission.Source = "department_default"
		}
		resolved = append(resolved, permission)
	}

	return resolved, nil
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
