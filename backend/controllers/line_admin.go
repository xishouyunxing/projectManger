package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"crane-system/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetLineAdminAssignments 返回产线管理员分配列表。
// 系统管理员看到全部，产线管理员只看到自己负责的。
func GetLineAdminAssignments(c *gin.Context) {
	role, _ := c.Get("user_role")
	userID := c.GetUint("user_id")

	query := database.DB.Preload("User").Preload("User.Department").Preload("ProductionLine")

	if role == "line_admin" {
		query = query.Where("user_id = ?", userID)
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

// CreateLineAdminAssignment 分配产线管理员。
// 系统管理员可分配所有产线，产线管理员只能分配自己负责的产线。
func CreateLineAdminAssignment(c *gin.Context) {
	currentRole, _ := c.Get("user_role")
	currentUserID := c.GetUint("user_id")

	var req struct {
		UserID           uint `json:"user_id" binding:"required"`
		ProductionLineID uint `json:"production_line_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 产线管理员只能分配自己负责的产线
	if currentRole == "line_admin" {
		if !services.IsLineManager(currentUserID, req.ProductionLineID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权操作该产线"})
			return
		}
	}

	// 检查目标用户角色是否为 line_admin
	var targetUser models.User
	if err := database.DB.First(&targetUser, req.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
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

// DeleteLineAdminAssignment 取消产线管理员分配。
func DeleteLineAdminAssignment(c *gin.Context) {
	assignmentID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	currentRole, _ := c.Get("user_role")
	currentUserID := c.GetUint("user_id")

	var assignment models.LineAdminAssignment
	if err := database.DB.First(&assignment, assignmentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分配记录不存在"})
		return
	}

	// 产线管理员只能取消自己负责产线的分配
	if currentRole == "line_admin" {
		if !services.IsLineManager(currentUserID, assignment.ProductionLineID) {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权操作该产线"})
			return
		}
	}

	affectedUserID := assignment.UserID
	if err := database.DB.Delete(&assignment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	services.InvalidateUserCache(affectedUserID)
	c.JSON(http.StatusOK, gin.H{"message": "已取消分配"})
}

// GetLinePermissionsByLine 产线管理员查看负责产线的用户权限。
func GetLinePermissionsByLine(c *gin.Context) {
	lineID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的产线ID"})
		return
	}

	currentRole, _ := c.Get("user_role")
	currentUserID := c.GetUint("user_id")

	// 产线管理员只能查看自己负责的产线
	if currentRole == "line_admin" {
		if !services.IsLineManager(currentUserID, uint(lineID)) {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权查看该产线"})
			return
		}
	}

	// 获取该产线的所有用户权限覆盖
	var userPerms []models.UserPermission
	database.DB.Preload("User").Preload("User.Department").
		Where("production_line_id = ?", lineID).Find(&userPerms)

	// 获取角色产线权限
	var rolePerms []models.RoleLinePermission
	database.DB.Preload("ProductionLine").
		Where("production_line_id = ?", lineID).Find(&rolePerms)

	c.JSON(http.StatusOK, gin.H{
		"production_line_id": lineID,
		"user_permissions":   userPerms,
		"role_permissions":   rolePerms,
	})
}

// SaveLinePermissionByAdmin 产线管理员为用户设置产线权限。
// 产线管理员只能分配 view/download/upload，不能分配 manage。
func SaveLinePermissionByAdmin(c *gin.Context) {
	lineID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的产线ID"})
		return
	}

	currentRole, _ := c.Get("user_role")
	currentUserID := c.GetUint("user_id")

	// 产线管理员只能操作自己负责的产线
	if currentRole == "line_admin" {
		if !services.IsLineManager(currentUserID, uint(lineID)) {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权操作该产线"})
			return
		}
	}

	var req struct {
		UserID      uint `json:"user_id" binding:"required"`
		CanView     bool `json:"can_view"`
		CanDownload bool `json:"can_download"`
		CanUpload   bool `json:"can_upload"`
		// 产线管理员不能分配 manage 权限
		CanManage *bool `json:"can_manage"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 产线管理员不能分配 manage
	if currentRole == "line_admin" && req.CanManage != nil && *req.CanManage {
		c.JSON(http.StatusForbidden, gin.H{"error": "产线管理员不能分配管理权限"})
		return
	}

	canManage := false
	if req.CanManage != nil {
		canManage = *req.CanManage
	}

	// upsert
	var existing models.UserPermission
	err = database.DB.Where("user_id = ? AND production_line_id = ?", req.UserID, lineID).
		First(&existing).Error
	if err != nil {
		database.DB.Create(&models.UserPermission{
			UserID:           req.UserID,
			ProductionLineID: uint(lineID),
			CanView:          req.CanView,
			CanDownload:      req.CanDownload,
			CanUpload:        req.CanUpload,
			CanManage:        canManage,
		})
	} else {
		database.DB.Model(&existing).Updates(map[string]interface{}{
			"can_view":     req.CanView,
			"can_download": req.CanDownload,
			"can_upload":   req.CanUpload,
			"can_manage":   canManage,
		})
	}

	services.InvalidateUserCache(req.UserID)
	c.JSON(http.StatusOK, gin.H{"message": "权限保存成功"})
}
