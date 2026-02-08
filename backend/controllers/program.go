package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetPrograms(c *gin.Context) {
	var programs []models.Program
	query := database.DB.Preload("ProductionLine").Preload("VehicleModel")

	if lineID := c.Query("production_line_id"); lineID != "" {
		query = query.Where("production_line_id = ?", lineID)
	}
	if vehicleID := c.Query("vehicle_model_id"); vehicleID != "" {
		query = query.Where("vehicle_model_id = ?", vehicleID)
	}

	if err := query.Find(&programs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, programs)
}

func GetProgram(c *gin.Context) {
	var program models.Program
	if err := database.DB.
		Preload("ProductionLine").
		Preload("VehicleModel").
		Preload("Files").
		Preload("Versions").
		First(&program, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
		return
	}

	c.JSON(http.StatusOK, program)
}

func CreateProgram(c *gin.Context) {
	var program models.Program
	if err := c.ShouldBindJSON(&program); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Create(&program).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusCreated, program)
}

func UpdateProgram(c *gin.Context) {
	var program models.Program
	if err := database.DB.First(&program, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
		return
	}

	if err := c.ShouldBindJSON(&program); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Save(&program).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, program)
}

func DeleteProgram(c *gin.Context) {
	if err := database.DB.Delete(&models.Program{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func GetProgramsByVehicle(c *gin.Context) {
	var programs []models.Program
	if err := database.DB.
		Preload("ProductionLine").
		Preload("ProductionLine.Process").
		Where("vehicle_model_id = ?", c.Param("vehicle_id")).
		Find(&programs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, programs)
}

func GetProgramRelations(c *gin.Context) {
	var relations []models.ProgramRelation
	if err := database.DB.
		Preload("SourceProgram").
		Preload("RelatedProgram").
		Where("source_program_id = ? OR related_program_id = ?", c.Param("program_id"), c.Param("program_id")).
		Find(&relations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, relations)
}

func CreateRelation(c *gin.Context) {
	var relation models.ProgramRelation
	if err := c.ShouldBindJSON(&relation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Create(&relation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusCreated, relation)
}

func DeleteRelation(c *gin.Context) {
	if err := database.DB.Delete(&models.ProgramRelation{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
