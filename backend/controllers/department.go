package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetDepartments(c *gin.Context) {
	var departments []models.Department
	query := database.DB

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Find(&departments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取部门列表失败，请稍后重试"})
		return
	}

	c.JSON(http.StatusOK, departments)
}

func GetDepartment(c *gin.Context) {
	var department models.Department
	if err := database.DB.First(&department, c.Param("id")).Error; err != nil {
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

func UpdateDepartment(c *gin.Context) {
	var department models.Department
	if err := database.DB.First(&department, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该部门信息"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请检查输入信息是否正确"})
		return
	}

	if err := database.DB.Model(&department).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新部门信息失败，请稍后重试"})
		return
	}

	c.JSON(http.StatusOK, department)
}

func DeleteDepartment(c *gin.Context) {
	if err := database.DB.Delete(&models.Department{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除部门失败，请稍后重试"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "部门已成功删除"})
}
