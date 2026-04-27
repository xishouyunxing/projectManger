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

func GetVehicleModels(c *gin.Context) {
	pageQuery := c.Query("page")
	pageSizeQuery := c.Query("page_size")
	paged := pageQuery != "" || pageSizeQuery != ""
	scope := strings.TrimSpace(c.Query("scope"))

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

	var vehicleModels []models.VehicleModel
	query := database.DB
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
				c.JSON(http.StatusOK, []models.VehicleModel{})
				return
			}
			c.JSON(http.StatusOK, gin.H{"items": []models.VehicleModel{}, "total": 0, "page": page, "page_size": pageSize})
			return
		}

		if scope == "selector" {
			visibleModelIDs := database.DB.Raw(`
				SELECT programs.vehicle_model_id AS id
				FROM programs
				WHERE programs.production_line_id IN ?
					AND programs.vehicle_model_id <> 0
					AND programs.deleted_at IS NULL
				UNION
				SELECT vehicle_models.id AS id
				FROM vehicle_models
				WHERE vehicle_models.deleted_at IS NULL
					AND NOT EXISTS (
						SELECT 1
						FROM programs
						WHERE programs.vehicle_model_id = vehicle_models.id
							AND programs.deleted_at IS NULL
					)
			`, lineIDs)
			query = query.Where("id IN (?)", visibleModelIDs)
		} else {
			subQuery := database.DB.Model(&models.Program{}).Select("DISTINCT vehicle_model_id").Where("production_line_id IN ? AND vehicle_model_id <> 0", lineIDs)
			query = query.Where("id IN (?)", subQuery)
		}
	}

	if series := strings.TrimSpace(c.Query("series")); series != "" {
		query = query.Where("series = ?", series)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		likeKeyword := "%" + strings.ToLower(keyword) + "%"
		query = query.Where("LOWER(COALESCE(name, '')) LIKE ? OR LOWER(COALESCE(code, '')) LIKE ?", likeKeyword, likeKeyword)
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
		if err := query.Model(&models.VehicleModel{}).Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
			return
		}
		query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	}

	if err := query.Find(&vehicleModels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	if !paged {
		c.JSON(http.StatusOK, vehicleModels)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     vehicleModels,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func GetVehicleModel(c *gin.Context) {
	vehicleModelID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "车型ID格式错误"})
		return
	}

	allowedLineIDs, statusCode, message := resolveAuthorizedLineIDs(c, lineActionView)
	if statusCode != 0 {
		c.JSON(statusCode, gin.H{"error": message})
		return
	}

	query := database.DB
	if allowedLineIDs != nil {
		lineIDs := make([]uint, 0, len(allowedLineIDs))
		for lineID := range allowedLineIDs {
			lineIDs = append(lineIDs, lineID)
		}
		if len(lineIDs) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "vehicle model not found"})
			return
		}
		query = query.Preload("Programs", "production_line_id IN ?", lineIDs)
	} else {
		query = query.Preload("Programs")
	}

	var vehicleModel models.VehicleModel
	if err := query.First(&vehicleModel, vehicleModelID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "vehicle model not found"})
		return
	}

	if allowedLineIDs != nil && len(vehicleModel.Programs) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "vehicle model not found"})
		return
	}

	c.JSON(http.StatusOK, vehicleModel)
}

func CreateVehicleModel(c *gin.Context) {
	var vehicleModel models.VehicleModel
	if err := c.ShouldBindJSON(&vehicleModel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Create(&vehicleModel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusCreated, vehicleModel)
}

type updateVehicleModelRequest struct {
	Name        *string `json:"name"`
	Code        *string `json:"code"`
	Series      *string `json:"series"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
}

func UpdateVehicleModel(c *gin.Context) {
	vehicleModelID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "车型ID格式错误"})
		return
	}

	var vehicleModel models.VehicleModel
	if err := database.DB.First(&vehicleModel, vehicleModelID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "车型不存在"})
		return
	}

	var req updateVehicleModelRequest
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
	if req.Series != nil {
		updates["series"] = strings.TrimSpace(*req.Series)
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != nil {
		updates["status"] = strings.TrimSpace(*req.Status)
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未提供可更新字段"})
		return
	}

	if err := database.DB.Model(&vehicleModel).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	if err := database.DB.First(&vehicleModel, vehicleModelID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, vehicleModel)
}

func DeleteVehicleModel(c *gin.Context) {
	vehicleModelID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "车型ID格式错误"})
		return
	}

	if dependency, err := findMasterDataDependency([]masterDataDependencyCheck{
		{Model: &models.Program{}, Where: "vehicle_model_id = ?", Args: []any{vehicleModelID}, Label: "programs"},
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "dependency check failed"})
		return
	} else if dependency != "" {
		c.JSON(http.StatusConflict, gin.H{"error": "vehicle model is in use by " + dependency})
		return
	}

	result := database.DB.Delete(&models.VehicleModel{}, vehicleModelID)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "车型不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
