package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetVehicleModels(c *gin.Context) {
	var vehicleModels []models.VehicleModel
	query := database.DB

	if series := c.Query("series"); series != "" {
		query = query.Where("series = ?", series)
	}

	if err := query.Find(&vehicleModels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, vehicleModels)
}

func GetVehicleModel(c *gin.Context) {
	var vehicleModel models.VehicleModel
	if err := database.DB.Preload("Programs").First(&vehicleModel, c.Param("id")).Error; err != nil {
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

func UpdateVehicleModel(c *gin.Context) {
	var vehicleModel models.VehicleModel
	if err := database.DB.First(&vehicleModel, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "车型不存在"})
		return
	}

	if err := c.ShouldBindJSON(&vehicleModel); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Save(&vehicleModel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, vehicleModel)
}

func DeleteVehicleModel(c *gin.Context) {
	if err := database.DB.Delete(&models.VehicleModel{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
