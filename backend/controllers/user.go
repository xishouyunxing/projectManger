package controllers

import (
	"crane-system/config"
	"crane-system/database"
	"crane-system/models"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func GetUsers(c *gin.Context) {
	pageQuery := c.Query("page")
	pageSizeQuery := c.Query("page_size")
	paged := pageQuery != "" || pageSizeQuery != ""

	page := 1
	if pageQuery != "" {
		parsedPage, err := strconv.Atoi(pageQuery)
		if err != nil || parsedPage < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page"})
			return
		}
		page = parsedPage
	}

	pageSize := 20
	if pageSizeQuery != "" {
		parsedPageSize, err := strconv.Atoi(pageSizeQuery)
		if err != nil || parsedPageSize < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page_size"})
			return
		}
		if parsedPageSize > 200 {
			parsedPageSize = 200
		}
		pageSize = parsedPageSize
	}

	var users []models.User
	query := database.DB.Preload("Department")

	if deptID := c.Query("department_id"); deptID != "" {
		query = query.Where("department_id = ?", deptID)
	}
	if role := strings.TrimSpace(c.Query("role")); role != "" {
		query = query.Where("role = ?", role)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		likeKeyword := "%" + strings.ToLower(keyword) + "%"
		query = query.Where("LOWER(COALESCE(employee_id, '')) LIKE ? OR LOWER(COALESCE(name, '')) LIKE ?", likeKeyword, likeKeyword)
	}
	if dateFrom := strings.TrimSpace(c.Query("date_from")); dateFrom != "" {
		parsedDate, err := time.Parse("2006-01-02", dateFrom)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date_from"})
			return
		}
		query = query.Where("created_at >= ?", parsedDate)
	}
	if dateTo := strings.TrimSpace(c.Query("date_to")); dateTo != "" {
		parsedDate, err := time.Parse("2006-01-02", dateTo)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date_to"})
			return
		}
		query = query.Where("created_at < ?", parsedDate.AddDate(0, 0, 1))
	}

	var total int64
	if paged {
		if err := query.Model(&models.User{}).Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
			return
		}
		query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	}

	if err := query.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	if !paged {
		c.JSON(http.StatusOK, users)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func GetUser(c *gin.Context) {
	targetID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID格式错误"})
		return
	}
	if !authorizeOwnerOrAdmin(c, targetID) {
		return
	}

	var user models.User
	if err := database.DB.Preload("Department").First(&user, targetID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}
	user.Password = string(hashedPassword)

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func UpdateUser(c *gin.Context) {
	targetID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "???ID??????"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, targetID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "????????"})
		return
	}

	userID, _ := c.Get("user_id")
	userRole, _ := c.Get("user_role")
	if userRole != "admin" && userID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "?????"})
		return
	}

	var payload map[string]json.RawMessage
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates, err := buildUserUpdates(payload, userRole == "admin")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "????????????"})
		return
	}

	if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "??????"})
		return
	}

	if err := database.DB.Preload("Department").First(&user, targetID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "??????"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func buildUserUpdates(payload map[string]json.RawMessage, isAdmin bool) (map[string]interface{}, error) {
	updates := map[string]interface{}{}

	for field, raw := range payload {
		switch field {
		case "name":
			value, err := parseJSONStringField(raw, field)
			if err != nil {
				return nil, err
			}
			if value == "" {
				return nil, invalidUserFieldValue(field)
			}
			updates["name"] = value
		case "employee_id":
			if !isAdmin {
				continue
			}
			value, err := parseJSONStringField(raw, field)
			if err != nil {
				return nil, err
			}
			if value == "" {
				return nil, invalidUserFieldValue(field)
			}
			updates["employee_id"] = value
		case "department_id":
			if !isAdmin {
				continue
			}
			if string(raw) == "null" {
				updates["department_id"] = nil
				continue
			}
			var value uint
			if err := json.Unmarshal(raw, &value); err != nil || value == 0 {
				return nil, invalidUserFieldValue(field)
			}
			updates["department_id"] = value
		case "role":
			if !isAdmin {
				continue
			}
			value, err := parseJSONStringField(raw, field)
			if err != nil {
				return nil, err
			}
			if value != "admin" && value != "user" {
				return nil, invalidUserFieldValue(field)
			}
			updates["role"] = value
		case "status":
			if !isAdmin {
				continue
			}
			value, err := parseJSONStringField(raw, field)
			if err != nil {
				return nil, err
			}
			if value != "active" && value != "inactive" {
				return nil, invalidUserFieldValue(field)
			}
			updates["status"] = value
		case "password", "id", "created_at", "updated_at", "deleted_at":
			continue
		default:
			continue
		}
	}

	return updates, nil
}

func parseJSONStringField(raw json.RawMessage, field string) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("%s ??????", field)
	}
	return strings.TrimSpace(value), nil
}

func invalidUserFieldValue(field string) error {
	return fmt.Errorf("%s ??????", field)
}

func DeleteUser(c *gin.Context) {
	targetID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID格式错误"})
		return
	}

	result := database.DB.Delete(&models.User{}, targetID)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password" binding:"required"`
}

func ChangePassword(c *gin.Context) {
	targetID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID格式错误"})
		return
	}

	if !authorizeOwnerOrAdmin(c, targetID) {
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	currentUserIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证用户"})
		return
	}
	currentUserID, ok := currentUserIDValue.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户身份无效"})
		return
	}
	isSelfService := currentUserID == targetID

	var user models.User
	if err := database.DB.First(&user, targetID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	if isSelfService {
		if req.OldPassword == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "旧密码不能为空"})
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "旧密码错误"})
			return
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	user.Password = string(hashedPassword)
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新密码失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "密码修改成功"})
}

func ResetPassword(c *gin.Context) {
	targetID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID格式错误"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, targetID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 重置为默认密码
	defaultPassword := config.AppConfig.Auth.DefaultPassword
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	user.Password = string(hashedPassword)
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重置密码失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "密码已重置为配置中的默认密码"})
}
