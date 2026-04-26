package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

	if err := validateUserPermissionRelations(permission.UserID, permission.ProductionLineID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing models.UserPermission
	err := database.DB.Where("user_id = ? AND production_line_id = ?", permission.UserID, permission.ProductionLineID).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	if err == nil {
		existing.CanView = permission.CanView
		existing.CanDownload = permission.CanDownload
		existing.CanUpload = permission.CanUpload
		existing.CanManage = permission.CanManage
		if err := database.DB.Save(&existing).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
			return
		}
		if err := database.DB.Preload("User").Preload("User.Department").Preload("ProductionLine").First(&existing, existing.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
			return
		}
		c.JSON(http.StatusOK, existing)
		return
	}

	if err := database.DB.Model(&models.UserPermission{}).Create(map[string]any{
		"user_id":            permission.UserID,
		"production_line_id": permission.ProductionLineID,
		"can_view":           permission.CanView,
		"can_download":       permission.CanDownload,
		"can_upload":         permission.CanUpload,
		"can_manage":         permission.CanManage,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	if err := database.DB.Preload("User").Preload("User.Department").Preload("ProductionLine").Where("user_id = ? AND production_line_id = ?", permission.UserID, permission.ProductionLineID).First(&permission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	c.JSON(http.StatusCreated, permission)
}

func validateUserPermissionRelations(userID, productionLineID uint) error {
	if userID == 0 {
		return errors.New("invalid user")
	}
	if productionLineID == 0 {
		return errors.New("invalid production line")
	}

	var user models.User
	if err := database.DB.Select("id").First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("invalid user")
		}
		return err
	}

	var line models.ProductionLine
	if err := database.DB.Select("id").First(&line, productionLineID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("invalid production line")
		}
		return err
	}

	return nil
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

	result := database.DB.Unscoped().Delete(&models.UserPermission{}, permissionID)
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
