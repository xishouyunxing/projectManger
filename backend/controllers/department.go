package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetDepartments(c *gin.Context) {
	pageQuery := c.Query("page")
	pageSizeQuery := c.Query("page_size")
	paged := pageQuery != "" || pageSizeQuery != ""

	page := 1
	if pageQuery != "" {
		parsedPage, err := strconv.Atoi(pageQuery)
		if err != nil || parsedPage < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "page参数格式错误"})
			return
		}
		page = parsedPage
	}

	pageSize := 20
	if pageSizeQuery != "" {
		parsedPageSize, err := strconv.Atoi(pageSizeQuery)
		if err != nil || parsedPageSize < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "page_size参数格式错误"})
			return
		}
		if parsedPageSize > 200 {
			parsedPageSize = 200
		}
		pageSize = parsedPageSize
	}

	var departments []models.Department
	query := database.DB

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if paged {
		if err := query.Model(&models.Department{}).Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取部门列表失败，请稍后重试"})
			return
		}
		query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	}

	if err := query.Find(&departments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取部门列表失败，请稍后重试"})
		return
	}

	if !paged {
		c.JSON(http.StatusOK, departments)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     departments,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func GetDepartment(c *gin.Context) {
	departmentID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "部门ID格式错误"})
		return
	}

	var department models.Department
	if err := database.DB.First(&department, departmentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该部门信息"})
		return
	}

	c.JSON(http.StatusOK, department)
}

func CreateDepartment(c *gin.Context) {
	var department models.Department
	if err := c.ShouldBindJSON(&department); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请检查输入信息是否完整"})
		return
	}

	if err := database.DB.Create(&department).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建部门失败，请检查部门名称是否重复"})
		return
	}

	c.JSON(http.StatusCreated, department)
}

type updateDepartmentRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
}

func UpdateDepartment(c *gin.Context) {
	departmentID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "部门ID格式错误"})
		return
	}

	var department models.Department
	if err := database.DB.First(&department, departmentID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该部门信息"})
		return
	}

	var req updateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请检查输入信息是否正确"})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid name"})
			return
		}
		updates["name"] = name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != nil {
		status := strings.TrimSpace(*req.Status)
		if status != "active" && status != "inactive" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
			return
		}
		updates["status"] = status
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未提供可更新字段"})
		return
	}

	if err := database.DB.Model(&department).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新部门信息失败，请稍后重试"})
		return
	}

	c.JSON(http.StatusOK, department)
}

func DeleteDepartment(c *gin.Context) {
	departmentID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "部门ID格式错误"})
		return
	}

	if dependency, err := findMasterDataDependency([]masterDataDependencyCheck{
		{Model: &models.User{}, Where: "department_id = ?", Args: []any{departmentID}, Label: "users"},
		{Model: &models.DepartmentPermission{}, Where: "department_id = ?", Args: []any{departmentID}, Label: "department permissions"},
		{Model: &models.DepartmentDefaultPermission{}, Where: "department_id = ?", Args: []any{departmentID}, Label: "department default permissions"},
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "dependency check failed"})
		return
	} else if dependency != "" {
		c.JSON(http.StatusConflict, gin.H{"error": "department is in use by " + dependency})
		return
	}

	result := database.DB.Delete(&models.Department{}, departmentID)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除部门失败，请稍后重试"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该部门信息"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "部门已成功删除"})
}
