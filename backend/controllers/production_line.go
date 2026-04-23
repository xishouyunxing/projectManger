package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)
func GetProductionLines(c *gin.Context) {
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

	var lines []models.ProductionLine
	query := database.DB.Preload("Process")

	if lineType := c.Query("type"); lineType != "" {
		query = query.Where("type = ?", lineType)
	}
	if processID := c.Query("process_id"); processID != "" {
		query = query.Where("process_id = ?", processID)
	}

	var total int64
	if paged {
		if err := query.Model(&models.ProductionLine{}).Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
			return
		}
		query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	}

	if err := query.Find(&lines).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	if !paged {
		c.JSON(http.StatusOK, lines)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     lines,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func GetProductionLine(c *gin.Context) {
	lineID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "生产线ID格式错误"})
		return
	}

	var line models.ProductionLine
	if err := database.DB.Preload("Process").Preload("Programs").First(&line, lineID).Error; err != nil {
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

type updateProductionLineRequest struct {
	Name        *string `json:"name"`
	Code        *string `json:"code"`
	Type        *string `json:"type"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	ProcessID   *uint   `json:"process_id"`
}

func UpdateProductionLine(c *gin.Context) {
	lineID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "生产线ID格式错误"})
		return
	}

	var line models.ProductionLine
	if err := database.DB.First(&line, lineID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "生产线不存在"})
		return
	}

	var req updateProductionLineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Code != nil {
		updates["code"] = strings.TrimSpace(*req.Code)
	}
	if req.Type != nil {
		updates["type"] = strings.TrimSpace(*req.Type)
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != nil {
		updates["status"] = strings.TrimSpace(*req.Status)
	}
	if req.ProcessID != nil {
		updates["process_id"] = req.ProcessID
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未提供可更新字段"})
		return
	}

	if err := database.DB.Model(&line).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	if err := database.DB.Preload("Process").First(&line, lineID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, line)
}

func DeleteProductionLine(c *gin.Context) {
	lineID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "生产线ID格式错误"})
		return
	}

	result := database.DB.Delete(&models.ProductionLine{}, lineID)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "生产线不存在"})
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
	processID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "工序ID格式错误"})
		return
	}

	var process models.Process
	if err := database.DB.Preload("ProductionLines").First(&process, processID).Error; err != nil {
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

type updateProcessRequest struct {
	Name        *string `json:"name"`
	Code        *string `json:"code"`
	Type        *string `json:"type"`
	SortOrder   *int    `json:"sort_order"`
	Description *string `json:"description"`
}

func UpdateProcess(c *gin.Context) {
	processID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "工序ID格式错误"})
		return
	}

	var process models.Process
	if err := database.DB.First(&process, processID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "工序不存在"})
		return
	}

	var req updateProcessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Code != nil {
		updates["code"] = strings.TrimSpace(*req.Code)
	}
	if req.Type != nil {
		updates["type"] = strings.TrimSpace(*req.Type)
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未提供可更新字段"})
		return
	}

	if err := database.DB.Model(&process).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	if err := database.DB.First(&process, processID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, process)
}

func DeleteProcess(c *gin.Context) {
	processID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "工序ID格式错误"})
		return
	}

	result := database.DB.Delete(&models.Process{}, processID)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "工序不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
