package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"net/http"
	"strconv"
	"strings"
	"time"

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

	var lines []models.ProductionLine
	query := database.DB.Preload("Process")
	allowedLineIDs, statusCode, message := resolveAuthorizedLineIDs(c, lineActionView)
	if statusCode != 0 {
		c.JSON(statusCode, gin.H{"error": message})
		return
	}
	if allowedLineIDs != nil {
		lineIDs := make([]uint, 0, len(allowedLineIDs))
		for lineID := range allowedLineIDs {
			lineIDs = append(lineIDs, lineID)
		}
		if len(lineIDs) == 0 {
			if !paged {
				c.JSON(http.StatusOK, []models.ProductionLine{})
				return
			}
			c.JSON(http.StatusOK, gin.H{"items": []models.ProductionLine{}, "total": 0, "page": page, "page_size": pageSize})
			return
		}
		query = query.Where("id IN ?", lineIDs)
	}

	if lineType := strings.TrimSpace(c.Query("type")); lineType != "" {
		query = query.Where("type = ?", lineType)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if processID := c.Query("process_id"); processID != "" {
		query = query.Where("process_id = ?", processID)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		likeKeyword := "%" + strings.ToLower(keyword) + "%"
		query = query.Where("LOWER(COALESCE(name, '')) LIKE ? OR LOWER(COALESCE(code, '')) LIKE ? OR LOWER(COALESCE(description, '')) LIKE ?", likeKeyword, likeKeyword, likeKeyword)
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
		if err := query.Model(&models.ProductionLine{}).Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
			return
		}
		query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	}

	if err := query.Find(&lines).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
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
	if !authorizeLineAction(c, line.ID, lineActionView) {
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

	if dependency, err := findMasterDataDependency([]masterDataDependencyCheck{
		{Model: &models.Program{}, Where: "production_line_id = ?", Args: []any{lineID}, Label: "programs"},
		{Model: &models.ProductionLineCustomField{}, Where: "production_line_id = ?", Args: []any{lineID}, Label: "custom fields"},
		{Model: &models.UserPermission{}, Where: "production_line_id = ?", Args: []any{lineID}, Label: "user permissions"},
		{Model: &models.DepartmentPermission{}, Where: "production_line_id = ?", Args: []any{lineID}, Label: "department permissions"},
		{Model: &models.RoleDefaultPermission{}, Where: "production_line_id = ?", Args: []any{lineID}, Label: "role default permissions"},
		{Model: &models.DepartmentDefaultPermission{}, Where: "production_line_id = ?", Args: []any{lineID}, Label: "department default permissions"},
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "dependency check failed"})
		return
	} else if dependency != "" {
		c.JSON(http.StatusConflict, gin.H{"error": "production line is in use by " + dependency})
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

	if dependency, err := findMasterDataDependency([]masterDataDependencyCheck{
		{Model: &models.ProductionLine{}, Where: "process_id = ?", Args: []any{processID}, Label: "production lines"},
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "dependency check failed"})
		return
	} else if dependency != "" {
		c.JSON(http.StatusConflict, gin.H{"error": "process is in use by " + dependency})
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
