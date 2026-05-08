package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"crane-system/services"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var user models.User
	if err := database.DB.Select("id").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user"})
		return
	}

	lines, err := loadPermissionMatrixLines()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	bits, err := services.LoadSubjectLinePermissionBits(services.PermissionSubject{Type: models.PermissionSubjectUser, ID: userID}, lines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	permissionMap := map[uint]services.LinePermissionBits{}
	for _, permission := range bits {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
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
		return services.SavePermissionRuleChangesTx(
			tx,
			services.PermissionSubject{Type: models.PermissionSubjectUser, ID: userID},
			permissionRuleChangesFromMatrixItems(req.Permissions, false),
		)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save failed"})
		return
	}
	services.InvalidateAllCache()

	c.JSON(http.StatusOK, gin.H{"message": "save succeeded"})
}

func GetDepartmentPermissionMatrix(c *gin.Context) {
	departmentID, err := parseUintParam(c.Param("department_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department id"})
		return
	}
	if err := validateDepartmentExists(departmentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lines, err := loadPermissionMatrixLines()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	bits, err := services.LoadSubjectLinePermissionBits(services.PermissionSubject{Type: models.PermissionSubjectDepartment, ID: departmentID}, lines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	permissionMap := map[uint]services.LinePermissionBits{}
	for _, permission := range bits {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department id"})
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
		return services.SavePermissionRuleChangesTx(
			tx,
			services.PermissionSubject{Type: models.PermissionSubjectDepartment, ID: departmentID},
			permissionRuleChangesFromMatrixItems(req.Permissions, false),
		)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save failed"})
		return
	}
	services.InvalidateAllCache()

	c.JSON(http.StatusOK, gin.H{"message": "save succeeded"})
}

func GetRoleDefaultPermissionMatrix(c *gin.Context) {
	role := strings.TrimSpace(c.Param("role"))
	if err := validateRoleValue(role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lines, err := loadPermissionMatrixLines()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	bits, err := services.LoadSubjectLinePermissionBits(services.PermissionSubject{Type: models.PermissionSubjectRole, Key: role}, lines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	permissionMap := map[uint]services.LinePermissionBits{}
	for _, permission := range bits {
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
		return services.SavePermissionRuleChangesTx(
			tx,
			services.PermissionSubject{Type: models.PermissionSubjectRole, Key: role},
			permissionRuleChangesFromMatrixItems(req.Permissions, true),
		)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save failed"})
		return
	}
	services.InvalidateAllCache()

	c.JSON(http.StatusOK, gin.H{"message": "save succeeded"})
}

func GetDepartmentDefaultPermissionMatrix(c *gin.Context) {
	departmentID, err := parseUintParam(c.Param("department_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department id"})
		return
	}
	if err := validateDepartmentExists(departmentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lines, err := loadPermissionMatrixLines()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	bits, err := services.LoadSubjectLinePermissionBits(services.PermissionSubject{Type: models.PermissionSubjectDepartmentDefault, ID: departmentID}, lines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	permissionMap := map[uint]services.LinePermissionBits{}
	for _, permission := range bits {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department id"})
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
		return services.SavePermissionRuleChangesTx(
			tx,
			services.PermissionSubject{Type: models.PermissionSubjectDepartmentDefault, ID: departmentID},
			permissionRuleChangesFromMatrixItems(req.Permissions, true),
		)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save failed"})
		return
	}
	services.InvalidateAllCache()

	c.JSON(http.StatusOK, gin.H{"message": "save succeeded"})
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

func permissionRuleChangesFromMatrixItems(items []savePermissionMatrixItem, emptyMeansUnset bool) []services.PermissionRuleChange {
	changes := make([]services.PermissionRuleChange, 0, len(items)*4)
	for _, item := range items {
		unset := item.Inherit || (emptyMeansUnset && permissionMatrixItemEmpty(item.CanView, item.CanDownload, item.CanUpload, item.CanManage))
		changes = append(changes,
			permissionRuleChangeForAction(item.ProductionLineID, models.PermissionActionView, item.CanView, unset),
			permissionRuleChangeForAction(item.ProductionLineID, models.PermissionActionDownload, item.CanDownload, unset),
			permissionRuleChangeForAction(item.ProductionLineID, models.PermissionActionUpload, item.CanUpload, unset),
			permissionRuleChangeForAction(item.ProductionLineID, models.PermissionActionManage, item.CanManage, unset),
		)
	}
	return changes
}
