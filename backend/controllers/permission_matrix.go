package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type permissionMatrixItem struct {
	ProductionLineID   uint   `json:"production_line_id"`
	ProductionLineName string `json:"production_line_name"`
	CanView            bool   `json:"can_view"`
	CanDownload        bool   `json:"can_download"`
	CanUpload          bool   `json:"can_upload"`
	CanManage          bool   `json:"can_manage"`
	Source             string `json:"source"`
}

type permissionMatrixResponse struct {
	OwnerType string                 `json:"owner_type"`
	OwnerID   uint                   `json:"owner_id,omitempty"`
	Role      string                 `json:"role,omitempty"`
	Items     []permissionMatrixItem `json:"items"`
}

type savePermissionMatrixRequest struct {
	Permissions []savePermissionMatrixItem `json:"permissions"`
}

type savePermissionMatrixItem struct {
	ProductionLineID uint `json:"production_line_id" binding:"required"`
	CanView          bool `json:"can_view"`
	CanDownload      bool `json:"can_download"`
	CanUpload        bool `json:"can_upload"`
	CanManage        bool `json:"can_manage"`
}

func GetUserPermissionMatrix(c *gin.Context) {
	userID, err := parseUintParam(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID格式错误"})
		return
	}

	var user models.User
	if err := database.DB.Select("id").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user"})
		return
	}

	lines, err := loadPermissionMatrixLines()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	var permissions []models.UserPermission
	if err := database.DB.Where("user_id = ?", userID).Find(&permissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	permissionMap := map[uint]models.UserPermission{}
	for _, permission := range permissions {
		permissionMap[permission.ProductionLineID] = permission
	}

	c.JSON(http.StatusOK, permissionMatrixResponse{
		OwnerType: "user",
		OwnerID:   userID,
		Items: buildPermissionMatrixItems(lines, func(lineID uint) (bool, bool, bool, bool, string) {
			if permission, ok := permissionMap[lineID]; ok {
				return permission.CanView, permission.CanDownload, permission.CanUpload, permission.CanManage, "user"
			}
			return false, false, false, false, "none"
		}),
	})
}

func SaveUserPermissionMatrix(c *gin.Context) {
	userID, err := parseUintParam(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID格式错误"})
		return
	}
	if err := validateUserExists(userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req savePermissionMatrixRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validatePermissionMatrixLines(req.Permissions); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range req.Permissions {
			if err := upsertUserPermissionOverride(tx, userID, item); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "保存成功"})
}

func GetDepartmentPermissionMatrix(c *gin.Context) {
	departmentID, err := parseUintParam(c.Param("department_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "部门ID格式错误"})
		return
	}
	if err := validateDepartmentExists(departmentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lines, err := loadPermissionMatrixLines()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	var permissions []models.DepartmentPermission
	if err := database.DB.Where("department_id = ?", departmentID).Find(&permissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	permissionMap := map[uint]models.DepartmentPermission{}
	for _, permission := range permissions {
		permissionMap[permission.ProductionLineID] = permission
	}

	c.JSON(http.StatusOK, permissionMatrixResponse{
		OwnerType: "department",
		OwnerID:   departmentID,
		Items: buildPermissionMatrixItems(lines, func(lineID uint) (bool, bool, bool, bool, string) {
			if permission, ok := permissionMap[lineID]; ok {
				return permission.CanView, permission.CanDownload, permission.CanUpload, permission.CanManage, "department"
			}
			return false, false, false, false, "none"
		}),
	})
}

func SaveDepartmentPermissionMatrix(c *gin.Context) {
	departmentID, err := parseUintParam(c.Param("department_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "部门ID格式错误"})
		return
	}
	if err := validateDepartmentExists(departmentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req savePermissionMatrixRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validatePermissionMatrixLines(req.Permissions); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range req.Permissions {
			if err := upsertDepartmentPermissionOverride(tx, departmentID, item); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "保存成功"})
}

func GetRoleDefaultPermissionMatrix(c *gin.Context) {
	role := strings.TrimSpace(c.Param("role"))
	if err := validateRoleValue(role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lines, err := loadPermissionMatrixLines()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	var permissions []models.RoleDefaultPermission
	if err := database.DB.Where("role = ?", role).Find(&permissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	permissionMap := map[uint]models.RoleDefaultPermission{}
	for _, permission := range permissions {
		permissionMap[permission.ProductionLineID] = permission
	}

	c.JSON(http.StatusOK, permissionMatrixResponse{
		OwnerType: "role_default",
		Role:      role,
		Items: buildPermissionMatrixItems(lines, func(lineID uint) (bool, bool, bool, bool, string) {
			if permission, ok := permissionMap[lineID]; ok {
				return permission.CanView, permission.CanDownload, permission.CanUpload, permission.CanManage, "role_default"
			}
			return false, false, false, false, "none"
		}),
	})
}

func SaveRoleDefaultPermissionMatrix(c *gin.Context) {
	role := strings.TrimSpace(c.Param("role"))
	if err := validateRoleValue(role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req savePermissionMatrixRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validatePermissionMatrixLines(req.Permissions); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range req.Permissions {
			if permissionMatrixItemEmpty(item) {
				if err := tx.Unscoped().Where("role = ? AND production_line_id = ?", role, item.ProductionLineID).Delete(&models.RoleDefaultPermission{}).Error; err != nil {
					return err
				}
				continue
			}

			var permission models.RoleDefaultPermission
			err := tx.Where("role = ? AND production_line_id = ?", role, item.ProductionLineID).First(&permission).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			permission.Role = role
			permission.ProductionLineID = item.ProductionLineID
			permission.CanView = item.CanView
			permission.CanDownload = item.CanDownload
			permission.CanUpload = item.CanUpload
			permission.CanManage = item.CanManage
			if err := tx.Save(&permission).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "保存成功"})
}

func GetDepartmentDefaultPermissionMatrix(c *gin.Context) {
	departmentID, err := parseUintParam(c.Param("department_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "部门ID格式错误"})
		return
	}
	if err := validateDepartmentExists(departmentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lines, err := loadPermissionMatrixLines()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	var permissions []models.DepartmentDefaultPermission
	if err := database.DB.Where("department_id = ?", departmentID).Find(&permissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	permissionMap := map[uint]models.DepartmentDefaultPermission{}
	for _, permission := range permissions {
		permissionMap[permission.ProductionLineID] = permission
	}

	c.JSON(http.StatusOK, permissionMatrixResponse{
		OwnerType: "department_default",
		OwnerID:   departmentID,
		Items: buildPermissionMatrixItems(lines, func(lineID uint) (bool, bool, bool, bool, string) {
			if permission, ok := permissionMap[lineID]; ok {
				return permission.CanView, permission.CanDownload, permission.CanUpload, permission.CanManage, "department_default"
			}
			return false, false, false, false, "none"
		}),
	})
}

func SaveDepartmentDefaultPermissionMatrix(c *gin.Context) {
	departmentID, err := parseUintParam(c.Param("department_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "部门ID格式错误"})
		return
	}
	if err := validateDepartmentExists(departmentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req savePermissionMatrixRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validatePermissionMatrixLines(req.Permissions); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range req.Permissions {
			if permissionMatrixItemEmpty(item) {
				if err := tx.Unscoped().Where("department_id = ? AND production_line_id = ?", departmentID, item.ProductionLineID).Delete(&models.DepartmentDefaultPermission{}).Error; err != nil {
					return err
				}
				continue
			}

			var permission models.DepartmentDefaultPermission
			err := tx.Where("department_id = ? AND production_line_id = ?", departmentID, item.ProductionLineID).First(&permission).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			permission.DepartmentID = departmentID
			permission.ProductionLineID = item.ProductionLineID
			permission.CanView = item.CanView
			permission.CanDownload = item.CanDownload
			permission.CanUpload = item.CanUpload
			permission.CanManage = item.CanManage
			if err := tx.Save(&permission).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "保存成功"})
}

func loadPermissionMatrixLines() ([]models.ProductionLine, error) {
	var lines []models.ProductionLine
	if err := database.DB.Order("id ASC").Find(&lines).Error; err != nil {
		return nil, err
	}
	return lines, nil
}

func buildPermissionMatrixItems(lines []models.ProductionLine, resolve func(uint) (bool, bool, bool, bool, string)) []permissionMatrixItem {
	items := make([]permissionMatrixItem, 0, len(lines))
	for _, line := range lines {
		canView, canDownload, canUpload, canManage, source := resolve(line.ID)
		items = append(items, permissionMatrixItem{
			ProductionLineID:   line.ID,
			ProductionLineName: line.Name,
			CanView:            canView,
			CanDownload:        canDownload,
			CanUpload:          canUpload,
			CanManage:          canManage,
			Source:             source,
		})
	}
	return items
}

func validateUserExists(userID uint) error {
	if userID == 0 {
		return errors.New("invalid user")
	}
	var user models.User
	if err := database.DB.Select("id").First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("invalid user")
		}
		return err
	}
	return nil
}

func validateDepartmentExists(departmentID uint) error {
	if departmentID == 0 {
		return errors.New("invalid department")
	}
	var department models.Department
	if err := database.DB.Select("id").First(&department, departmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("invalid department")
		}
		return err
	}
	return nil
}

func validateRoleValue(role string) error {
	if role == "" {
		return errors.New("invalid role")
	}
	if role == "admin" || role == "user" {
		return nil
	}
	var count int64
	if err := database.DB.Model(&models.User{}).Where("role = ?", role).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("invalid role")
	}
	return nil
}

func validatePermissionMatrixLines(items []savePermissionMatrixItem) error {
	seen := map[uint]struct{}{}
	for _, item := range items {
		if item.ProductionLineID == 0 {
			return errors.New("invalid production line")
		}
		if _, ok := seen[item.ProductionLineID]; ok {
			return errors.New("duplicate production line")
		}
		seen[item.ProductionLineID] = struct{}{}

		var line models.ProductionLine
		if err := database.DB.Select("id").First(&line, item.ProductionLineID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("invalid production line")
			}
			return err
		}
	}
	return nil
}

func permissionMatrixItemEmpty(item savePermissionMatrixItem) bool {
	return !item.CanView && !item.CanDownload && !item.CanUpload && !item.CanManage
}

func permissionMatrixUpdates(item savePermissionMatrixItem) map[string]any {
	return map[string]any{
		"can_view":     item.CanView,
		"can_download": item.CanDownload,
		"can_upload":   item.CanUpload,
		"can_manage":   item.CanManage,
	}
}

func upsertUserPermissionOverride(tx *gorm.DB, userID uint, item savePermissionMatrixItem) error {
	updates := permissionMatrixUpdates(item)
	var count int64
	if err := tx.Model(&models.UserPermission{}).
		Where("user_id = ? AND production_line_id = ?", userID, item.ProductionLineID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return tx.Model(&models.UserPermission{}).
			Where("user_id = ? AND production_line_id = ?", userID, item.ProductionLineID).
			Updates(updates).Error
	}

	updates["user_id"] = userID
	updates["production_line_id"] = item.ProductionLineID
	return tx.Model(&models.UserPermission{}).Create(updates).Error
}

func upsertDepartmentPermissionOverride(tx *gorm.DB, departmentID uint, item savePermissionMatrixItem) error {
	updates := permissionMatrixUpdates(item)
	var count int64
	if err := tx.Model(&models.DepartmentPermission{}).
		Where("department_id = ? AND production_line_id = ?", departmentID, item.ProductionLineID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return tx.Model(&models.DepartmentPermission{}).
			Where("department_id = ? AND production_line_id = ?", departmentID, item.ProductionLineID).
			Updates(updates).Error
	}

	updates["department_id"] = departmentID
	updates["production_line_id"] = item.ProductionLineID
	return tx.Model(&models.DepartmentPermission{}).Create(updates).Error
}
