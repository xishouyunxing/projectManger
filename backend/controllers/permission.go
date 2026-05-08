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

func GetPermissions(c *gin.Context) {
	var users []models.User
	query := database.DB.Preload("Department")
	if userID, err := parseOptionalUintQuery(c.Query("user_id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user"})
		return
	} else if userID != 0 {
		query = query.Where("id = ?", userID)
	}
	if err := query.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	lines, ok := loadFilteredPermissionLines(c.Query("production_line_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid production line"})
		return
	}
	permissions := make([]models.UserPermission, 0)
	for _, user := range users {
		bits, err := services.LoadSubjectLinePermissionBits(services.PermissionSubject{Type: models.PermissionSubjectUser, ID: user.ID}, lines)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
			return
		}
		for _, bit := range bits {
			permissions = append(permissions, userPermissionFromBits(user, bit, lines))
		}
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

	status := http.StatusCreated
	if _, exists, err := services.LoadSubjectLinePermissionBitsByLine(services.PermissionSubject{Type: models.PermissionSubjectUser, ID: permission.UserID}, permission.ProductionLineID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	} else if exists {
		status = http.StatusOK
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		return syncLinePermissionRulesTx(tx, services.PermissionSubject{Type: models.PermissionSubjectUser, ID: permission.UserID}, linePermissionBits{
			ProductionLineID: permission.ProductionLineID,
			CanView:          permission.CanView,
			CanDownload:      permission.CanDownload,
			CanUpload:        permission.CanUpload,
			CanManage:        permission.CanManage,
		})
	}); err != nil {
		if status == http.StatusOK {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		}
		return
	}
	services.InvalidateAllCache()

	response, ok, err := loadRuleBackedUserPermission(permission.UserID, permission.ProductionLineID)
	if err != nil || !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	c.JSON(status, response)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid permission id"})
		return
	}

	var permission models.UserPermission
	if err := loadLegacyUserPermissionByID(permissionID, &permission); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "permission not found"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "no updatable fields"})
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		applyUserPermissionUpdates(&permission, updates)
		return syncLinePermissionRulesTx(tx, services.PermissionSubject{Type: models.PermissionSubjectUser, ID: permission.UserID}, linePermissionBits{
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

	response, ok, err := loadRuleBackedUserPermission(permission.UserID, permission.ProductionLineID)
	if err != nil || !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	c.JSON(http.StatusOK, response)
}

func DeletePermission(c *gin.Context) {
	permissionID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid permission id"})
		return
	}

	var permission models.UserPermission
	if err := loadLegacyUserPermissionByID(permissionID, &permission); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "permission not found"})
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		return clearLinePermissionRulesTx(tx, services.PermissionSubject{Type: models.PermissionSubjectUser, ID: permission.UserID}, permission.ProductionLineID)
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	services.InvalidateAllCache()

	c.JSON(http.StatusOK, gin.H{"message": "delete succeeded"})
}

func GetUserPermissions(c *gin.Context) {
	targetID, err := parseUintParam(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	if !authorizeOwnerOrAdmin(c, targetID) {
		return
	}

	var user models.User
	if err := database.DB.Preload("Department").First(&user, targetID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	lines, err := loadPermissionMatrixLines()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	bits, err := services.LoadSubjectLinePermissionBits(services.PermissionSubject{Type: models.PermissionSubjectUser, ID: targetID}, lines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	permissions := make([]models.UserPermission, 0, len(bits))
	for _, bit := range bits {
		permissions = append(permissions, userPermissionFromBits(user, bit, lines))
	}

	c.JSON(http.StatusOK, permissions)
}

func parseOptionalUintQuery(value string) (uint, error) {
	if value == "" {
		return 0, nil
	}
	return parseUintParam(value)
}

func loadFilteredPermissionLines(lineIDQuery string) ([]models.ProductionLine, bool) {
	var lines []models.ProductionLine
	query := database.DB.Preload("Process").Order("id ASC")
	if lineIDQuery != "" {
		lineID, err := parseUintParam(lineIDQuery)
		if err != nil {
			return nil, false
		}
		query = query.Where("id = ?", lineID)
	}
	return lines, query.Find(&lines).Error == nil
}

func userPermissionFromBits(user models.User, bits services.LinePermissionBits, lines []models.ProductionLine) models.UserPermission {
	permission := models.UserPermission{
		ID:               syntheticLinePermissionID(user.ID, bits.ProductionLineID),
		UserID:           user.ID,
		ProductionLineID: bits.ProductionLineID,
		CanView:          bits.CanView,
		CanDownload:      bits.CanDownload,
		CanUpload:        bits.CanUpload,
		CanManage:        bits.CanManage,
		User:             user,
	}
	for _, line := range lines {
		if line.ID == bits.ProductionLineID {
			permission.ProductionLine = line
			break
		}
	}
	return permission
}

func syntheticLinePermissionID(ownerID, lineID uint) uint {
	return ownerID*1000000 + lineID
}

func loadRuleBackedUserPermission(userID, lineID uint) (models.UserPermission, bool, error) {
	var user models.User
	if err := database.DB.Preload("Department").First(&user, userID).Error; err != nil {
		return models.UserPermission{}, false, err
	}
	var lines []models.ProductionLine
	if err := database.DB.Preload("Process").Where("id = ?", lineID).Find(&lines).Error; err != nil {
		return models.UserPermission{}, false, err
	}
	if len(lines) == 0 {
		return models.UserPermission{}, false, nil
	}
	bits, exists, err := services.LoadSubjectLinePermissionBitsByLine(services.PermissionSubject{Type: models.PermissionSubjectUser, ID: userID}, lineID)
	if err != nil || !exists {
		return models.UserPermission{}, false, err
	}
	return userPermissionFromBits(user, bits, lines), true, nil
}

func loadLegacyUserPermissionByID(permissionID uint, permission *models.UserPermission) error {
	var users []models.User
	if err := database.DB.Select("id").Find(&users).Error; err != nil {
		return err
	}
	lines, err := loadPermissionMatrixLines()
	if err != nil {
		return err
	}
	for _, user := range users {
		bits, err := services.LoadSubjectLinePermissionBits(services.PermissionSubject{Type: models.PermissionSubjectUser, ID: user.ID}, lines)
		if err != nil {
			return err
		}
		for _, bit := range bits {
			if syntheticLinePermissionID(user.ID, bit.ProductionLineID) == permissionID {
				*permission = models.UserPermission{
					ID:               permissionID,
					UserID:           user.ID,
					ProductionLineID: bit.ProductionLineID,
					CanView:          bit.CanView,
					CanDownload:      bit.CanDownload,
					CanUpload:        bit.CanUpload,
					CanManage:        bit.CanManage,
				}
				return nil
			}
		}
	}
	return gorm.ErrRecordNotFound
}

func applyUserPermissionUpdates(permission *models.UserPermission, updates map[string]interface{}) {
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
