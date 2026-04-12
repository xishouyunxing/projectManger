package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetDepartmentPermissions(c *gin.Context) {
	var permissions []models.DepartmentPermission
	query := database.DB.Preload("Department").Preload("ProductionLine")

	if deptID := c.Query("department_id"); deptID != "" {
		query = query.Where("department_id = ?", deptID)
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

func CreateDepartmentPermission(c *gin.Context) {
	var permission models.DepartmentPermission
	if err := c.ShouldBindJSON(&permission); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing models.DepartmentPermission
	err := database.DB.Where("department_id = ? AND production_line_id = ?", permission.DepartmentID, permission.ProductionLineID).First(&existing).Error
	if err == nil {
		existing.CanView = permission.CanView
		existing.CanDownload = permission.CanDownload
		existing.CanUpload = permission.CanUpload
		existing.CanManage = permission.CanManage
		if err := database.DB.Save(&existing).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新权限失败"})
			return
		}
		c.JSON(http.StatusOK, existing)
		return
	}

	if err := database.DB.Create(&permission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建权限失败"})
		return
	}

	c.JSON(http.StatusCreated, permission)
}

func UpdateDepartmentPermission(c *gin.Context) {
	var permission models.DepartmentPermission
	if err := database.DB.First(&permission, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "权限不存在"})
		return
	}

	if err := c.ShouldBindJSON(&permission); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Save(&permission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, permission)
}

func DeleteDepartmentPermission(c *gin.Context) {
	if err := database.DB.Unscoped().Delete(&models.DepartmentPermission{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func GetUserEffectivePermissions(c *gin.Context) {
	userID := c.Param("user_id")

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	var productionLines []models.ProductionLine
	if err := database.DB.Find(&productionLines).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	var userPermissions []models.UserPermission
	database.DB.Where("user_id = ?", userID).Find(&userPermissions)
	userPermMap := make(map[uint]models.UserPermission)
	for _, perm := range userPermissions {
		userPermMap[perm.ProductionLineID] = perm
	}

	var deptPermissions []models.DepartmentPermission
	if user.DepartmentID != nil {
		database.DB.Where("department_id = ?", *user.DepartmentID).Find(&deptPermissions)
	}
	deptPermMap := make(map[uint]models.DepartmentPermission)
	for _, perm := range deptPermissions {
		deptPermMap[perm.ProductionLineID] = perm
	}

	type EffectivePermission struct {
		ProductionLineID   uint   `json:"production_line_id"`
		ProductionLineName string `json:"production_line_name"`
		CanView            bool   `json:"can_view"`
		CanDownload        bool   `json:"can_download"`
		CanUpload          bool   `json:"can_upload"`
		CanManage          bool   `json:"can_manage"`
		Source             string `json:"source"`
	}

	var effectivePermissions []EffectivePermission
	for _, line := range productionLines {
		userPerm, hasUserPerm := userPermMap[line.ID]
		deptPerm, hasDeptPerm := deptPermMap[line.ID]

		ep := EffectivePermission{
			ProductionLineID:   line.ID,
			ProductionLineName: line.Name,
		}

		if hasUserPerm && hasDeptPerm {
			ep.CanView = userPerm.CanView || deptPerm.CanView
			ep.CanDownload = userPerm.CanDownload || deptPerm.CanDownload
			ep.CanUpload = userPerm.CanUpload || deptPerm.CanUpload
			ep.CanManage = userPerm.CanManage || deptPerm.CanManage
			ep.Source = "both"
		} else if hasUserPerm {
			ep.CanView = userPerm.CanView
			ep.CanDownload = userPerm.CanDownload
			ep.CanUpload = userPerm.CanUpload
			ep.CanManage = userPerm.CanManage
			ep.Source = "user"
		} else if hasDeptPerm {
			ep.CanView = deptPerm.CanView
			ep.CanDownload = deptPerm.CanDownload
			ep.CanUpload = deptPerm.CanUpload
			ep.CanManage = deptPerm.CanManage
			ep.Source = "department"
		} else {
			ep.Source = "none"
		}

		effectivePermissions = append(effectivePermissions, ep)
	}

	c.JSON(http.StatusOK, effectivePermissions)
}
