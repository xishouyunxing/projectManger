package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

	if err := validateDepartmentPermissionRelations(permission.DepartmentID, permission.ProductionLineID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing models.DepartmentPermission
	err := database.DB.Where("department_id = ? AND production_line_id = ?", permission.DepartmentID, permission.ProductionLineID).First(&existing).Error
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

func validateDepartmentPermissionRelations(departmentID, productionLineID uint) error {
	if departmentID == 0 {
		return errors.New("invalid department")
	}
	if productionLineID == 0 {
		return errors.New("invalid production line")
	}

	var department models.Department
	if err := database.DB.Select("id").First(&department, departmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("invalid department")
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

type updateDepartmentPermissionRequest struct {
	CanView     *bool `json:"can_view"`
	CanDownload *bool `json:"can_download"`
	CanUpload   *bool `json:"can_upload"`
	CanManage   *bool `json:"can_manage"`
}

func UpdateDepartmentPermission(c *gin.Context) {
	permissionID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "权限ID格式错误"})
		return
	}

	var permission models.DepartmentPermission
	if err := database.DB.First(&permission, permissionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "权限不存在"})
		return
	}

	var req updateDepartmentPermissionRequest
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
	if err := database.DB.Preload("Department").Preload("ProductionLine").First(&permission, permissionID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, permission)
}

func DeleteDepartmentPermission(c *gin.Context) {
	permissionID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "权限ID格式错误"})
		return
	}

	result := database.DB.Unscoped().Delete(&models.DepartmentPermission{}, permissionID)
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

func GetUserEffectivePermissions(c *gin.Context) {
	targetID, err := parseUintParam(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID格式错误"})
		return
	}
	if !authorizeOwnerOrAdmin(c, targetID) {
		return
	}

	var user models.User
	if err := database.DB.First(&user, targetID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	var productionLines []models.ProductionLine
	if err := database.DB.Find(&productionLines).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	var userPermissions []models.UserPermission
	database.DB.Where("user_id = ?", targetID).Find(&userPermissions)
	userPermMap := make(map[uint]models.UserPermission)
	for _, perm := range userPermissions {
		existing := userPermMap[perm.ProductionLineID]
		existing.ProductionLineID = perm.ProductionLineID
		existing.CanView = existing.CanView || perm.CanView
		existing.CanDownload = existing.CanDownload || perm.CanDownload
		existing.CanUpload = existing.CanUpload || perm.CanUpload
		existing.CanManage = existing.CanManage || perm.CanManage
		userPermMap[perm.ProductionLineID] = existing
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
