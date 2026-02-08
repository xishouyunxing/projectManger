package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetProductionLines(c *gin.Context) {
	var lines []models.ProductionLine
	query := database.DB.Preload("Process")

	if lineType := c.Query("type"); lineType != "" {
		query = query.Where("type = ?", lineType)
	}
	if processID := c.Query("process_id"); processID != "" {
		query = query.Where("process_id = ?", processID)
	}

	if err := query.Find(&lines).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, lines)
}

func GetProductionLine(c *gin.Context) {
	var line models.ProductionLine
	if err := database.DB.Preload("Process").Preload("Programs").First(&line, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "生产线不存在"})
		return
	}

	c.JSON(http.StatusOK, line)
}

func CreateProductionLine(c *gin.Context) {
	var line models.ProductionLine
	if err := c.ShouldBindJSON(&line); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Create(&line).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusCreated, line)
}

func UpdateProductionLine(c *gin.Context) {
	var line models.ProductionLine
	if err := database.DB.First(&line, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "生产线不存在"})
		return
	}

	if err := c.ShouldBindJSON(&line); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Save(&line).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, line)
}

func DeleteProductionLine(c *gin.Context) {
	if err := database.DB.Delete(&models.ProductionLine{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func GetProcesses(c *gin.Context) {
	var processes []models.Process
	query := database.DB

	if processType := c.Query("type"); processType != "" {
		query = query.Where("type = ?", processType)
	}

	if err := query.Order("sort_order").Find(&processes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, processes)
}

func GetProcess(c *gin.Context) {
	var process models.Process
	if err := database.DB.Preload("ProductionLines").First(&process, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工序不存在"})
		return
	}

	c.JSON(http.StatusOK, process)
}

func CreateProcess(c *gin.Context) {
	var process models.Process
	if err := c.ShouldBindJSON(&process); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Create(&process).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusCreated, process)
}

func UpdateProcess(c *gin.Context) {
	var process models.Process
	if err := database.DB.First(&process, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工序不存在"})
		return
	}

	if err := c.ShouldBindJSON(&process); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Save(&process).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, process)
}

func DeleteProcess(c *gin.Context) {
	if err := database.DB.Delete(&models.Process{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
