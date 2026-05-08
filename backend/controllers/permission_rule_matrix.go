package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"crane-system/services"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type permissionRuleMatrixResponse struct {
	OwnerType string                          `json:"owner_type"`
	OwnerID   uint                            `json:"owner_id,omitempty"`
	OwnerKey  string                          `json:"owner_key,omitempty"`
	Items     []services.PermissionMatrixLine `json:"items"`
}

type savePermissionRuleRequest struct {
	Changes []services.PermissionRuleChange `json:"changes"`
}

func GetUserEffectivePermissionMatrix(c *gin.Context) {
	user, ok := loadPermissionMatrixUser(c)
	if !ok {
		return
	}
	lines, ok := loadPermissionRuleMatrixLines(c)
	if !ok {
		return
	}
	items, err := services.ResolveSubjectMatrix(user, services.PermissionSubject{Type: models.PermissionSubjectUser, ID: user.ID}, lines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询权限失败"})
		return
	}
	c.JSON(http.StatusOK, permissionRuleMatrixResponse{OwnerType: models.PermissionSubjectUser, OwnerID: user.ID, Items: items})
}

func SaveUserPermissionRules(c *gin.Context) {
	user, ok := loadPermissionMatrixUser(c)
	if !ok {
		return
	}
	savePermissionRulesAndReturnMatrix(c, services.PermissionSubject{Type: models.PermissionSubjectUser, ID: user.ID}, func(lines []models.ProductionLine) ([]services.PermissionMatrixLine, error) {
		return services.ResolveSubjectMatrix(user, services.PermissionSubject{Type: models.PermissionSubjectUser, ID: user.ID}, lines)
	})
}

func GetDepartmentEffectivePermissionMatrix(c *gin.Context) {
	departmentID, ok := loadPermissionMatrixDepartmentID(c)
	if !ok {
		return
	}
	returnSubjectRuleMatrix(c, services.PermissionSubject{Type: models.PermissionSubjectDepartment, ID: departmentID})
}

func SaveDepartmentPermissionRules(c *gin.Context) {
	departmentID, ok := loadPermissionMatrixDepartmentID(c)
	if !ok {
		return
	}
	savePermissionRulesAndReturnRuleMatrix(c, services.PermissionSubject{Type: models.PermissionSubjectDepartment, ID: departmentID})
}

func GetRoleEffectivePermissionMatrix(c *gin.Context) {
	subject, ok := loadPermissionMatrixRoleSubject(c)
	if !ok {
		return
	}
	returnSubjectRuleMatrix(c, subject)
}

func SaveRolePermissionRules(c *gin.Context) {
	subject, ok := loadPermissionMatrixRoleSubject(c)
	if !ok {
		return
	}
	savePermissionRulesAndReturnRuleMatrix(c, subject)
}

func GetDepartmentDefaultPermissionRuleMatrix(c *gin.Context) {
	departmentID, ok := loadPermissionMatrixDepartmentID(c)
	if !ok {
		return
	}
	returnSubjectRuleMatrix(c, services.PermissionSubject{Type: models.PermissionSubjectDepartmentDefault, ID: departmentID})
}

func SaveDepartmentDefaultPermissionRules(c *gin.Context) {
	departmentID, ok := loadPermissionMatrixDepartmentID(c)
	if !ok {
		return
	}
	savePermissionRulesAndReturnRuleMatrix(c, services.PermissionSubject{Type: models.PermissionSubjectDepartmentDefault, ID: departmentID})
}

func returnSubjectRuleMatrix(c *gin.Context, subject services.PermissionSubject) {
	lines, ok := loadPermissionRuleMatrixLines(c)
	if !ok {
		return
	}
	items, err := services.ResolveSubjectRuleMatrix(subject, lines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询权限失败"})
		return
	}
	c.JSON(http.StatusOK, permissionRuleMatrixResponse{OwnerType: subject.Type, OwnerID: subject.ID, OwnerKey: subject.Key, Items: items})
}

func savePermissionRulesAndReturnRuleMatrix(c *gin.Context, subject services.PermissionSubject) {
	savePermissionRulesAndReturnMatrix(c, subject, func(lines []models.ProductionLine) ([]services.PermissionMatrixLine, error) {
		return services.ResolveSubjectRuleMatrix(subject, lines)
	})
}

func savePermissionRulesAndReturnMatrix(c *gin.Context, subject services.PermissionSubject, resolve func([]models.ProductionLine) ([]services.PermissionMatrixLine, error)) {
	var req savePermissionRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validatePermissionRuleChanges(req.Changes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := services.SavePermissionRuleChanges(subject, req.Changes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	lines, ok := loadPermissionRuleMatrixLines(c)
	if !ok {
		return
	}
	items, err := resolve(lines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询权限失败"})
		return
	}
	c.JSON(http.StatusOK, permissionRuleMatrixResponse{OwnerType: subject.Type, OwnerID: subject.ID, OwnerKey: subject.Key, Items: items})
}

func validatePermissionRuleChanges(changes []services.PermissionRuleChange) error {
	seen := map[string]struct{}{}
	for _, change := range changes {
		resourceType := strings.TrimSpace(change.ResourceType)
		if resourceType == "" {
			resourceType = models.PermissionResourceProductionLine
		}
		if resourceType != models.PermissionResourceProductionLine {
			return errors.New("invalid resource type")
		}
		if change.ResourceID == 0 {
			return errors.New("invalid resource id")
		}
		key := resourceType + ":" + strconv.FormatUint(uint64(change.ResourceID), 10) + ":" + change.Action
		if _, ok := seen[key]; ok {
			return errors.New("duplicate change")
		}
		seen[key] = struct{}{}
		var line models.ProductionLine
		if err := database.DB.Select("id").First(&line, change.ResourceID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("invalid production line")
			}
			return err
		}
	}
	return nil
}

func loadPermissionMatrixUser(c *gin.Context) (models.User, bool) {
	userID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID格式错误"})
		return models.User{}, false
	}
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return models.User{}, false
	}
	return user, true
}

func loadPermissionMatrixDepartmentID(c *gin.Context) (uint, bool) {
	departmentID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "部门ID格式错误"})
		return 0, false
	}
	if err := validateDepartmentExists(departmentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return 0, false
	}
	return departmentID, true
}

func loadPermissionMatrixRoleSubject(c *gin.Context) (services.PermissionSubject, bool) {
	roleParam := strings.TrimSpace(c.Param("role"))
	if roleParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "角色格式错误"})
		return services.PermissionSubject{}, false
	}
	if parsed, err := strconv.ParseUint(roleParam, 10, 64); err == nil && parsed > 0 {
		var role models.Role
		if err := database.DB.First(&role, uint(parsed)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
			return services.PermissionSubject{}, false
		}
		return services.PermissionSubject{Type: models.PermissionSubjectRole, ID: role.ID, Key: role.Name}, true
	}
	var role models.Role
	err := database.DB.Where("name = ?", roleParam).First(&role).Error
	if err == nil {
		return services.PermissionSubject{Type: models.PermissionSubjectRole, ID: role.ID, Key: role.Name}, true
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询角色失败"})
		return services.PermissionSubject{}, false
	}
	return services.PermissionSubject{Type: models.PermissionSubjectRole, Key: roleParam}, true
}

func loadPermissionRuleMatrixLines(c *gin.Context) ([]models.ProductionLine, bool) {
	lines, err := loadPermissionMatrixLines()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询产线失败"})
		return nil, false
	}
	return lines, true
}
