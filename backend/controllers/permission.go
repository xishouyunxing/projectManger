package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetPermissions(c *gin.Context) {
	var permissions []models.UserPermission
	query := database.DB.Preload("User").Preload("User.Department").Preload("ProductionLine")

	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if lineID := c.Query("production_line_id"); lineID != "" {
		query = query.Where("production_line_id = ?", lineID)
	}

	if err := query.Find(&permissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, permissions)
}

func CreatePermission(c *gin.Context) {
	var permission models.UserPermission
	if err := c.ShouldBindJSON(&permission); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Create(&permission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusCreated, permission)
}

type updatePermissionRequest struct {
	CanView     *bool `json:"can_view"`
	CanDownload *bool `json:"can_download"`
	CanUpload   *bool `json:"can_upload"`
	CanManage   *bool `json:"can_manage"`
}

func UpdatePermission(c *gin.Context) {
	permissionID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "权限ID格式错误"})
		return
	}

	var permission models.UserPermission
	if err := database.DB.First(&permission, permissionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "权限不存在"})
		return
	}

	var req updatePermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.CanView != nil {
		updates["can_view"] = *req.CanView
	}
	if req.CanDownload != nil {
		updates["can_download"] = *req.CanDownload
	}
	if req.CanUpload != nil {
		updates["can_upload"] = *req.CanUpload
	}
	if req.CanManage != nil {
		updates["can_manage"] = *req.CanManage
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未提供可更新字段"})
		return
	}

	if err := database.DB.Model(&permission).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	if err := database.DB.Preload("User").Preload("User.Department").Preload("ProductionLine").First(&permission, permissionID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, permission)
}

func DeletePermission(c *gin.Context) {
	permissionID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "权限ID格式错误"})
		return
	}

	result := database.DB.Delete(&models.UserPermission{}, permissionID)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "权限不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func GetUserPermissions(c *gin.Context) {
	targetID, err := parseUintParam(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID格式错误"})
		return
	}
	if !authorizeOwnerOrAdmin(c, targetID) {
		return
	}

	var permissions []models.UserPermission
	if err := database.DB.
		Preload("ProductionLine").
		Preload("ProductionLine.Process").
		Where("user_id = ?", targetID).
		Find(&permissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, permissions)
}
