package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"crane-system/services"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetDepartmentPermissions(c *gin.Context) {
	var departments []models.Department
	query := database.DB
	if departmentID, err := parseOptionalUintQuery(c.Query("department_id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid department"})
		return
	} else if departmentID != 0 {
		query = query.Where("id = ?", departmentID)
	}
	if err := query.Find(&departments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	lines, ok := loadFilteredPermissionLines(c.Query("production_line_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid production line"})
		return
	}
	permissions := make([]models.DepartmentPermission, 0)
	for _, department := range departments {
		bits, err := services.LoadSubjectLinePermissionBits(services.PermissionSubject{Type: models.PermissionSubjectDepartment, ID: department.ID}, lines)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
			return
		}
		for _, bit := range bits {
			permissions = append(permissions, departmentPermissionFromBits(department, bit, lines))
		}
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

	status := http.StatusCreated
	if _, exists, err := services.LoadSubjectLinePermissionBitsByLine(services.PermissionSubject{Type: models.PermissionSubjectDepartment, ID: permission.DepartmentID}, permission.ProductionLineID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	} else if exists {
		status = http.StatusOK
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		return syncLinePermissionRulesTx(tx, services.PermissionSubject{Type: models.PermissionSubjectDepartment, ID: permission.DepartmentID}, linePermissionBits{
			ProductionLineID: permission.ProductionLineID,
			CanView:          permission.CanView,
			CanDownload:      permission.CanDownload,
			CanUpload:        permission.CanUpload,
			CanManage:        permission.CanManage,
		})
	}); err != nil {
		if status == http.StatusOK {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "update permission failed"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "create permission failed"})
		}
		return
	}
	services.InvalidateAllCache()

	response, ok, err := loadRuleBackedDepartmentPermission(permission.DepartmentID, permission.ProductionLineID)
	if err != nil || !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	c.JSON(status, response)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid permission id"})
		return
	}

	var permission models.DepartmentPermission
	if err := loadLegacyDepartmentPermissionByID(permissionID, &permission); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "permission not found"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "no updatable fields"})
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		applyDepartmentPermissionUpdates(&permission, updates)
		return syncLinePermissionRulesTx(tx, services.PermissionSubject{Type: models.PermissionSubjectDepartment, ID: permission.DepartmentID}, linePermissionBits{
			ProductionLineID: permission.ProductionLineID,
			CanView:          permission.CanView,
			CanDownload:      permission.CanDownload,
			CanUpload:        permission.CanUpload,
			CanManage:        permission.CanManage,
		})
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	services.InvalidateAllCache()

	response, ok, err := loadRuleBackedDepartmentPermission(permission.DepartmentID, permission.ProductionLineID)
	if err != nil || !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	c.JSON(http.StatusOK, response)
}

func DeleteDepartmentPermission(c *gin.Context) {
	permissionID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid permission id"})
		return
	}

	var permission models.DepartmentPermission
	if err := loadLegacyDepartmentPermissionByID(permissionID, &permission); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "permission not found"})
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		return clearLinePermissionRulesTx(tx, services.PermissionSubject{Type: models.PermissionSubjectDepartment, ID: permission.DepartmentID}, permission.ProductionLineID)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	services.InvalidateAllCache()

	c.JSON(http.StatusOK, gin.H{"message": "delete succeeded"})
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

	type EffectivePermission struct {
		ProductionLineID   uint   `json:"production_line_id"`
		ProductionLineName string `json:"production_line_name"`
		CanView            bool   `json:"can_view"`
		CanDownload        bool   `json:"can_download"`
		CanUpload          bool   `json:"can_upload"`
		CanManage          bool   `json:"can_manage"`
		Source             string `json:"source"`
	}

	// 使用新的权限服务获取有效权限
	permData, err := services.GetUserPermissions(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询权限失败"})
		return
	}

	lineNames := make(map[uint]string, len(productionLines))
	for _, line := range productionLines {
		lineNames[line.ID] = line.Name
	}

	var effectivePermissions []EffectivePermission
	for _, line := range productionLines {
		ep := EffectivePermission{
			ProductionLineID:   line.ID,
			ProductionLineName: line.Name,
		}
		if lp, ok := permData.LinePermissions[line.ID]; ok {
			ep.CanView = lp.CanView
			ep.CanDownload = lp.CanDownload
			ep.CanUpload = lp.CanUpload
			ep.CanManage = lp.CanManage
			ep.Source = lp.Source
		} else {
			ep.Source = "none"
		}
		effectivePermissions = append(effectivePermissions, ep)
	}

	c.JSON(http.StatusOK, effectivePermissions)
}

func departmentPermissionFromBits(department models.Department, bits services.LinePermissionBits, lines []models.ProductionLine) models.DepartmentPermission {
	permission := models.DepartmentPermission{
		ID:               syntheticLinePermissionID(department.ID, bits.ProductionLineID),
		DepartmentID:     department.ID,
		ProductionLineID: bits.ProductionLineID,
		CanView:          bits.CanView,
		CanDownload:      bits.CanDownload,
		CanUpload:        bits.CanUpload,
		CanManage:        bits.CanManage,
		Department:       department,
	}
	for _, line := range lines {
		if line.ID == bits.ProductionLineID {
			permission.ProductionLine = line
			break
		}
	}
	return permission
}

func loadRuleBackedDepartmentPermission(departmentID, lineID uint) (models.DepartmentPermission, bool, error) {
	var department models.Department
	if err := database.DB.First(&department, departmentID).Error; err != nil {
		return models.DepartmentPermission{}, false, err
	}
	var lines []models.ProductionLine
	if err := database.DB.Preload("Process").Where("id = ?", lineID).Find(&lines).Error; err != nil {
		return models.DepartmentPermission{}, false, err
	}
	if len(lines) == 0 {
		return models.DepartmentPermission{}, false, nil
	}
	bits, exists, err := services.LoadSubjectLinePermissionBitsByLine(services.PermissionSubject{Type: models.PermissionSubjectDepartment, ID: departmentID}, lineID)
	if err != nil || !exists {
		return models.DepartmentPermission{}, false, err
	}
	return departmentPermissionFromBits(department, bits, lines), true, nil
}

func loadLegacyDepartmentPermissionByID(permissionID uint, permission *models.DepartmentPermission) error {
	departmentID, lineID, ok := services.DecodeSyntheticLinePermissionID(permissionID)
	if !ok {
		return gorm.ErrRecordNotFound
	}
	if err := validateDepartmentPermissionRelations(departmentID, lineID); err != nil {
		return gorm.ErrRecordNotFound
	}
	bits, exists, err := services.LoadSubjectLinePermissionBitsByLine(services.PermissionSubject{Type: models.PermissionSubjectDepartment, ID: departmentID}, lineID)
	if err != nil {
		return err
	}
	if !exists {
		return gorm.ErrRecordNotFound
	}
	*permission = models.DepartmentPermission{
		ID:               permissionID,
		DepartmentID:     departmentID,
		ProductionLineID: lineID,
		CanView:          bits.CanView,
		CanDownload:      bits.CanDownload,
		CanUpload:        bits.CanUpload,
		CanManage:        bits.CanManage,
	}
	return nil
}

func applyDepartmentPermissionUpdates(permission *models.DepartmentPermission, updates map[string]interface{}) {
	if v, ok := updates["can_view"].(bool); ok {
		permission.CanView = v
	}
	if v, ok := updates["can_download"].(bool); ok {
		permission.CanDownload = v
	}
	if v, ok := updates["can_upload"].(bool); ok {
		permission.CanUpload = v
	}
	if v, ok := updates["can_manage"].(bool); ok {
		permission.CanManage = v
	}
}
