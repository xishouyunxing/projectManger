package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)
func GetVehicleModels(c *gin.Context) {
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

	var vehicleModels []models.VehicleModel
	query := database.DB

	if series := c.Query("series"); series != "" {
		query = query.Where("series = ?", series)
	}

	var total int64
	if paged {
		if err := query.Model(&models.VehicleModel{}).Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
			return
		}
		query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	}

	if err := query.Find(&vehicleModels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
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

	var vehicleModel models.VehicleModel
	if err := database.DB.Preload("Programs").First(&vehicleModel, vehicleModelID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "车型不存在"})
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
