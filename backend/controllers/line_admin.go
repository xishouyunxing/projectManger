package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"crane-system/services"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func isSystemAdminRole(role string) bool {
	return role == "admin" || role == "system_admin"
}

func authorizeLineAdminScope(c *gin.Context, productionLineID uint) bool {
	roleValue, _ := c.Get("user_role")
	role, _ := roleValue.(string)
	if isSystemAdminRole(role) {
		return true
	}

	userID := c.GetUint("user_id")
	if role == "line_admin" && services.IsLineManager(userID, productionLineID) {
		return true
	}

	c.JSON(http.StatusForbidden, gin.H{"error": "无权操作该产线"})
	return false
}

func GetLineAdminAssignments(c *gin.Context) {
	roleValue, _ := c.Get("user_role")
	role, _ := roleValue.(string)
	userID := c.GetUint("user_id")

	query := database.DB.Preload("User").Preload("User.Department").Preload("ProductionLine")
	if !isSystemAdminRole(role) {
		if role != "line_admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权查看产线管理员分配"})
			return
		}

		perms, err := services.GetUserPermissions(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询权限失败"})
			return
		}
		if len(perms.ManagedLineIDs) == 0 {
			c.JSON(http.StatusOK, []models.LineAdminAssignment{})
			return
		}
		query = query.Where("production_line_id IN ?", perms.ManagedLineIDs)
	}

	if lineID := c.Query("production_line_id"); lineID != "" {
		query = query.Where("production_line_id = ?", lineID)
	}
	if targetUserID := c.Query("user_id"); targetUserID != "" {
		query = query.Where("user_id = ?", targetUserID)
	}

	var assignments []models.LineAdminAssignment
	if err := query.Find(&assignments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, assignments)
}

func CreateLineAdminAssignment(c *gin.Context) {
	currentRoleValue, _ := c.Get("user_role")
	currentRole, _ := currentRoleValue.(string)
	currentUserID := c.GetUint("user_id")

	var req struct {
		UserID           uint `json:"user_id" binding:"required"`
		ProductionLineID uint `json:"production_line_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if currentRole == "line_admin" && !services.IsLineManager(currentUserID, req.ProductionLineID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作该产线"})
		return
	}

	var targetUser models.User
	if err := database.DB.First(&targetUser, req.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	var line models.ProductionLine
	if err := database.DB.Select("id").First(&line, req.ProductionLineID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid production line"})
		return
	}

	assignment := models.LineAdminAssignment{
		UserID:           req.UserID,
		ProductionLineID: req.ProductionLineID,
		CreatedBy:        &currentUserID,
	}
	if err := database.DB.Create(&assignment).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "该分配已存在"})
		return
	}

	services.InvalidateUserCache(req.UserID)
	c.JSON(http.StatusOK, assignment)
}

func DeleteLineAdminAssignment(c *gin.Context) {
	assignmentID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	currentRoleValue, _ := c.Get("user_role")
	currentRole, _ := currentRoleValue.(string)
	currentUserID := c.GetUint("user_id")

	var assignment models.LineAdminAssignment
	if err := database.DB.First(&assignment, assignmentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分配记录不存在"})
		return
	}

	if currentRole == "line_admin" && !services.IsLineManager(currentUserID, assignment.ProductionLineID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作该产线"})
		return
	}

	affectedUserID := assignment.UserID
	if err := database.DB.Delete(&assignment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	services.InvalidateUserCache(affectedUserID)
	c.JSON(http.StatusOK, gin.H{"message": "已取消分配"})
}

func GetLinePermissionsByLine(c *gin.Context) {
	lineID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的产线ID"})
		return
	}
	if !authorizeLineAdminScope(c, uint(lineID)) {
		return
	}

	var userPerms []models.UserPermission
	database.DB.Preload("User").Preload("User.Department").
		Where("production_line_id = ?", lineID).Find(&userPerms)

	var rolePerms []models.RoleLinePermission
	database.DB.Preload("ProductionLine").
		Where("production_line_id = ?", lineID).Find(&rolePerms)

	c.JSON(http.StatusOK, gin.H{
		"production_line_id": lineID,
		"user_permissions":   userPerms,
		"role_permissions":   rolePerms,
	})
}

func SaveLinePermissionByAdmin(c *gin.Context) {
	lineID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的产线ID"})
		return
	}
	if !authorizeLineAdminScope(c, uint(lineID)) {
		return
	}

	roleValue, _ := c.Get("user_role")
	currentRole, _ := roleValue.(string)

	var req struct {
		UserID      uint  `json:"user_id" binding:"required"`
		CanView     bool  `json:"can_view"`
		CanDownload bool  `json:"can_download"`
		CanUpload   bool  `json:"can_upload"`
		CanManage   *bool `json:"can_manage"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if currentRole == "line_admin" && req.CanManage != nil && *req.CanManage {
		c.JSON(http.StatusForbidden, gin.H{"error": "产线管理员不能分配管理权限"})
		return
	}
	if err := validateUserPermissionRelations(req.UserID, uint(lineID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	canManage := false
	if req.CanManage != nil {
		canManage = *req.CanManage
	}
	bits := linePermissionBits{
		ProductionLineID: uint(lineID),
		CanView:          req.CanView,
		CanDownload:      req.CanDownload,
		CanUpload:        req.CanUpload,
		CanManage:        canManage,
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		var existing models.UserPermission
		err := tx.Where("user_id = ? AND production_line_id = ?", req.UserID, lineID).
			First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(&models.UserPermission{
				UserID:           req.UserID,
				ProductionLineID: uint(lineID),
				CanView:          req.CanView,
				CanDownload:      req.CanDownload,
				CanUpload:        req.CanUpload,
				CanManage:        canManage,
			}).Error; err != nil {
				return err
			}
		} else if err := tx.Model(&existing).Updates(map[string]interface{}{
			"can_view":     req.CanView,
			"can_download": req.CanDownload,
			"can_upload":   req.CanUpload,
			"can_manage":   canManage,
		}).Error; err != nil {
			return err
		}
		return syncLinePermissionRulesTx(tx, services.PermissionSubject{Type: models.PermissionSubjectUser, ID: req.UserID}, bits)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "权限保存失败"})
		return
	}
	services.InvalidateAllCache()
	c.JSON(http.StatusOK, gin.H{"message": "权限保存成功"})
}
