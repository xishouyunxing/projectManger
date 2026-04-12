package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"errors"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type programCustomFieldValueSummary struct {
	FieldID   uint   `json:"field_id"`
	FieldName string `json:"field_name"`
	FieldType string `json:"field_type"`
	SortOrder int    `json:"sort_order"`
	Value     string `json:"value"`
}

type programListItem struct {
	models.Program
	CustomFieldValues []programCustomFieldValueSummary `json:"custom_field_values"`
}

func GetPrograms(c *gin.Context) {
	var programs []models.Program
	query := database.DB.
		Preload("ProductionLine").
		Preload("VehicleModel").
		Preload("CustomFieldValues").
		Preload("CustomFieldValues.ProductionLineCustomField")

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

	response := make([]programListItem, 0, len(programs))
	for _, program := range programs {
		item := programListItem{Program: program}
		item.CustomFieldValues = summarizeEnabledProgramCustomFieldValues(program.CustomFieldValues)
		response = append(response, item)
	}

	c.JSON(http.StatusOK, response)
}

func summarizeEnabledProgramCustomFieldValues(values []models.ProgramCustomFieldValue) []programCustomFieldValueSummary {
	summaries := make([]programCustomFieldValueSummary, 0, len(values))
	for _, value := range values {
		field := value.ProductionLineCustomField
		if field.ID == 0 || !field.Enabled {
			continue
		}

		summaries = append(summaries, programCustomFieldValueSummary{
			FieldID:   field.ID,
			FieldName: field.Name,
			FieldType: field.FieldType,
			SortOrder: field.SortOrder,
			Value:     value.Value,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].SortOrder == summaries[j].SortOrder {
			return summaries[i].FieldID < summaries[j].FieldID
		}
		return summaries[i].SortOrder < summaries[j].SortOrder
	})

	return summaries
}

func GetProgram(c *gin.Context) {
	var program models.Program
	if err := database.DB.
		Preload("ProductionLine").
		Preload("VehicleModel").
		Preload("Files").
		Preload("Versions").
		Preload("CustomFieldValues").
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
	originalProductionLineID := program.ProductionLineID

	if err := c.ShouldBindJSON(&program); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&program).Error; err != nil {
			return err
		}
		if originalProductionLineID != program.ProductionLineID {
			if err := tx.Where("program_id = ?", program.ID).Delete(&models.ProgramCustomFieldValue{}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, program)
}

func DeleteProgram(c *gin.Context) {
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var program models.Program
		if err := tx.First(&program, c.Param("id")).Error; err != nil {
			return err
		}
		if err := tx.Where("program_id = ?", c.Param("id")).Delete(&models.ProgramCustomFieldValue{}).Error; err != nil {
			return err
		}
		return tx.Delete(&program).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
			return
		}
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
