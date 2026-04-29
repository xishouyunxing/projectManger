package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"crane-system/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetRoles 返回所有角色列表。
func GetRoles(c *gin.Context) {
	var roles []models.Role
	query := database.DB.Order("sort_order ASC, id ASC")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Find(&roles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询角色失败"})
		return
	}

	c.JSON(http.StatusOK, roles)
}

// GetRole 返回单个角色详情，包含功能权限和产线权限。
func GetRole(c *gin.Context) {
	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的角色ID"})
		return
	}

	var role models.Role
	if err := database.DB.First(&role, roleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	// 加载功能权限
	var rolePerms []models.RolePermission
	database.DB.Where("role_id = ?", roleID).Find(&rolePerms)

	permIDs := make([]uint, 0, len(rolePerms))
	for _, rp := range rolePerms {
		permIDs = append(permIDs, rp.PermissionID)
	}

	var permissions []models.Permission
	if len(permIDs) > 0 {
		database.DB.Where("id IN ?", permIDs).Find(&permissions)
	}

	// 加载产线权限
	var linePerms []models.RoleLinePermission
	database.DB.Preload("ProductionLine").Where("role_id = ?", roleID).Find(&linePerms)

	c.JSON(http.StatusOK, gin.H{
		"role":             role,
		"permissions":      permissions,
		"line_permissions": linePerms,
	})
}

// CreateRoleRequest 创建角色的请求体。
type CreateRoleRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

// CreateRole 创建新角色。
func CreateRole(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role := models.Role{
		Name:        req.Name,
		Description: req.Description,
		IsPreset:    false,
		IsSystem:    false,
		Status:      "active",
		SortOrder:   req.SortOrder,
	}

	if err := database.DB.Create(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建角色失败"})
		return
	}

	services.InvalidateAllCache()
	c.JSON(http.StatusOK, role)
}

// UpdateRoleRequest 更新角色的请求体。
type UpdateRoleRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	SortOrder   *int    `json:"sort_order"`
}

// UpdateRole 更新角色信息。系统角色不可改名。
func UpdateRole(c *gin.Context) {
	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的角色ID"})
		return
	}

	var role models.Role
	if err := database.DB.First(&role, roleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if role.IsSystem && req.Name != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "系统角色不可修改名称"})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&role).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新角色失败"})
			return
		}
	}

	services.InvalidateAllCache()
	c.JSON(http.StatusOK, role)
}

// DeleteRole 删除角色。预设角色和系统角色不可删除。
func DeleteRole(c *gin.Context) {
	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的角色ID"})
		return
	}

	var role models.Role
	if err := database.DB.First(&role, roleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	if role.IsPreset || role.IsSystem {
		c.JSON(http.StatusForbidden, gin.H{"error": "预设角色不可删除"})
		return
	}

	// 检查是否有用户使用该角色
	var count int64
	database.DB.Model(&models.User{}).Where("role_id = ?", roleID).Count(&count)
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "该角色下仍有用户，无法删除"})
		return
	}

	// 删除关联数据
	database.DB.Where("role_id = ?", roleID).Delete(&models.RolePermission{})
	database.DB.Where("role_id = ?", roleID).Delete(&models.RoleLinePermission{})

	if err := database.DB.Delete(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除角色失败"})
		return
	}

	services.InvalidateAllCache()
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// SaveRolePermissionsRequest 保存角色功能权限的请求体。
type SaveRolePermissionsRequest struct {
	PermissionIDs []uint `json:"permission_ids"`
}

// SaveRolePermissions 批量设置角色的功能权限（全量替换）。
func SaveRolePermissions(c *gin.Context) {
	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的角色ID"})
		return
	}

	var role models.Role
	if err := database.DB.First(&role, roleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	var req SaveRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 全量替换
	database.DB.Where("role_id = ?", roleID).Delete(&models.RolePermission{})

	for _, pid := range req.PermissionIDs {
		database.DB.Create(&models.RolePermission{
			RoleID:       uint(roleID),
			PermissionID: pid,
		})
	}

	services.InvalidateAllCache()
	c.JSON(http.StatusOK, gin.H{"message": "功能权限保存成功"})
}

// GetAllPermissions 返回所有可用的功能权限定义。
func GetAllPermissions(c *gin.Context) {
	var permissions []models.Permission
	if err := database.DB.Order("type ASC, code ASC").Find(&permissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询权限失败"})
		return
	}
	c.JSON(http.StatusOK, permissions)
}

// SaveRoleLinePermissionsRequest 保存角色产线权限的请求体。
type SaveRoleLinePermissionsRequest struct {
	Permissions []RoleLinePermItem `json:"permissions"`
}

// RoleLinePermItem 单条产线权限。
type RoleLinePermItem struct {
	ProductionLineID uint `json:"production_line_id"`
	CanView          bool `json:"can_view"`
	CanDownload      bool `json:"can_download"`
	CanUpload        bool `json:"can_upload"`
	CanManage        bool `json:"can_manage"`
}

// SaveRoleLinePermissions 保存角色的产线权限矩阵（增量更新）。
func SaveRoleLinePermissions(c *gin.Context) {
	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的角色ID"})
		return
	}

	var role models.Role
	if err := database.DB.First(&role, roleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	var req SaveRoleLinePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, item := range req.Permissions {
		allFalse := !item.CanView && !item.CanDownload && !item.CanUpload && !item.CanManage
		if allFalse {
			// 全部为 false 时删除记录
			database.DB.Where("role_id = ? AND production_line_id = ?", roleID, item.ProductionLineID).
				Delete(&models.RoleLinePermission{})
		} else {
			// upsert
			var existing models.RoleLinePermission
			err := database.DB.Where("role_id = ? AND production_line_id = ?", roleID, item.ProductionLineID).
				First(&existing).Error
			if err != nil {
				database.DB.Create(&models.RoleLinePermission{
					RoleID:           uint(roleID),
					ProductionLineID: item.ProductionLineID,
					CanView:          item.CanView,
					CanDownload:      item.CanDownload,
					CanUpload:        item.CanUpload,
					CanManage:        item.CanManage,
				})
			} else {
				database.DB.Model(&existing).Updates(map[string]interface{}{
					"can_view":     item.CanView,
					"can_download": item.CanDownload,
					"can_upload":   item.CanUpload,
					"can_manage":   item.CanManage,
				})
			}
		}
	}

	services.InvalidateAllCache()
	c.JSON(http.StatusOK, gin.H{"message": "产线权限保存成功"})
}

// GetRoleLinePermissions 获取角色的产线权限矩阵。
func GetRoleLinePermissions(c *gin.Context) {
	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的角色ID"})
		return
	}

	var role models.Role
	if err := database.DB.First(&role, roleID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	// 获取所有产线
	var lines []models.ProductionLine
	database.DB.Where("status = ?", "active").Order("id ASC").Find(&lines)

	// 获取角色产线权限
	var rolePerms []models.RoleLinePermission
	database.DB.Where("role_id = ?", roleID).Find(&rolePerms)
	rolePermMap := map[uint]models.RoleLinePermission{}
	for _, rp := range rolePerms {
		rolePermMap[rp.ProductionLineID] = rp
	}

	// 构建矩阵
	type MatrixRow struct {
		ProductionLineID   uint   `json:"production_line_id"`
		ProductionLineName string `json:"production_line_name"`
		CanView            bool   `json:"can_view"`
		CanDownload        bool   `json:"can_download"`
		CanUpload          bool   `json:"can_upload"`
		CanManage          bool   `json:"can_manage"`
	}

	rows := make([]MatrixRow, 0, len(lines))
	for _, line := range lines {
		row := MatrixRow{
			ProductionLineID:   line.ID,
			ProductionLineName: line.Name,
		}
		if rp, ok := rolePermMap[line.ID]; ok {
			row.CanView = rp.CanView
			row.CanDownload = rp.CanDownload
			row.CanUpload = rp.CanUpload
			row.CanManage = rp.CanManage
		}
		rows = append(rows, row)
	}

	c.JSON(http.StatusOK, gin.H{
		"role":       role,
		"lines":      lines,
		"permissions": rows,
	})
}
