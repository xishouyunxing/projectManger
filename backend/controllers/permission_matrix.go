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
	Override           bool   `json:"override"`
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

// savePermissionMatrixItem 是矩阵保存的最小变更单元。
// Inherit=true 表示清除显式覆盖并回到继承；Inherit=false 表示写入显式覆盖，四个权限位全 false 也必须保留为“显式拒绝”。
//
// 前端只会提交被管理员改过的行，后端也按“增量补丁”处理这里的数组。
// 不要把未提交的产线理解为 false 或删除，否则会把继承权限批量改写成显式配置。
type savePermissionMatrixItem struct {
	ProductionLineID uint `json:"production_line_id" binding:"required"`
	CanView          bool `json:"can_view"`
	CanDownload      bool `json:"can_download"`
	CanUpload        bool `json:"can_upload"`
	CanManage        bool `json:"can_manage"`
	Inherit          bool `json:"inherit"`
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

// SaveUserPermissionMatrix 只处理前端提交的脏行。
// 这样可以避免把未修改的继承权限固化成用户显式覆盖。
//
// 用户显式权限优先级最高，因此保存语义必须能区分三种状态：
// 1. 不提交该产线：保持现状；
// 2. inherit=true：删除用户显式覆盖，重新回落到部门、角色默认或部门默认；
// 3. inherit=false：写入用户显式覆盖，即使四个权限位全 false 也代表明确拒绝。
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
			// inherit 是“清除覆盖”的操作，不是把权限位保存为 false。
			// 删除显式记录后，最终授权结果会由下级继承链重新计算。
			if item.Inherit {
				if err := tx.Unscoped().Where("user_id = ? AND production_line_id = ?", userID, item.ProductionLineID).Delete(&models.UserPermission{}).Error; err != nil {
					return err
				}
				continue
			}
			// 非 inherit 的请求始终保存为显式覆盖。
			// permissionMatrixItemEmpty 在用户矩阵中不能用于删除，否则会破坏“显式拒绝”。
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

// SaveDepartmentPermissionMatrix 与用户矩阵一致：Inherit 清除部门覆盖，否则写入显式覆盖。
//
// 部门显式权限会影响部门内所有未被用户显式覆盖的账号，因此同样需要保留全 false 的显式拒绝。
// 只有 inherit=true 才允许删除部门覆盖，普通全 false 保存必须落库。
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
			// 清除部门覆盖后，最终授权会继续回落到角色默认或部门默认权限。
			if item.Inherit {
				if err := tx.Unscoped().Where("department_id = ? AND production_line_id = ?", departmentID, item.ProductionLineID).Delete(&models.DepartmentPermission{}).Error; err != nil {
					return err
				}
				continue
			}
			// 全 false 仍然是部门级显式拒绝，不能复用默认权限矩阵的“空权限删除”逻辑。
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

// buildPermissionMatrixItems 为前端统一补齐所有产线行。
// Override 表示该行当前是否来自显式配置；前端据此展示“继承/覆盖”模式。
//
// Source 用于告诉前端权限来源，Override 用于告诉前端是否允许清除覆盖。
// source=none 的行不代表最终无权限，只代表当前矩阵层级没有配置，最终权限可能由更低优先级来源决定。
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
			Override:           source != "none",
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

// permissionMatrixUpdates 只生成四个权限位的更新内容。
// 是否删除覆盖由调用方通过 Inherit 决定，不能在这里根据全 false 自动删除。
func permissionMatrixUpdates(item savePermissionMatrixItem) map[string]any {
	return map[string]any{
		"can_view":     item.CanView,
		"can_download": item.CanDownload,
		"can_upload":   item.CanUpload,
		"can_manage":   item.CanManage,
	}
}

// upsertUserPermissionOverride 保存用户层显式覆盖。
// 这里先 Count 再 Updates/Create，是为了兼容当前模型的软删除和唯一约束迁移状态；
// 即使 item 四个权限位全 false，也必须写入或更新记录来表达显式拒绝。
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

// upsertDepartmentPermissionOverride 保存部门层显式覆盖，语义与用户覆盖一致。
// 默认权限矩阵可以把空权限视为删除，但部门覆盖不能这样做，否则无法阻断继承权限。
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
