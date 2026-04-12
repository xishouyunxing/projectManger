package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type saveProgramCustomFieldValuesRequest struct {
	Values []programCustomFieldValueInput `json:"values"`
}

type programCustomFieldValueInput struct {
	FieldID uint   `json:"field_id"`
	Value   string `json:"value"`
}

func SaveProgramCustomFieldValues(c *gin.Context) {
	var req saveProgramCustomFieldValuesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Values == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errProgramCustomFieldValuesRequired.Error()})
		return
	}

	var savedValues []models.ProgramCustomFieldValue
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var program models.Program
		if err := tx.First(&program, c.Param("id")).Error; err != nil {
			return err
		}

		newValues := make([]models.ProgramCustomFieldValue, 0, len(req.Values))
		seenFieldIDs := make(map[uint]struct{}, len(req.Values))
		for _, input := range req.Values {
			if input.FieldID == 0 {
				return errProgramCustomFieldFieldIDRequired
			}
			if _, exists := seenFieldIDs[input.FieldID]; exists {
				return errProgramCustomFieldDuplicateFieldID
			}
			seenFieldIDs[input.FieldID] = struct{}{}

			var field models.ProductionLineCustomField
			if err := tx.Where("id = ? AND production_line_id = ?", input.FieldID, program.ProductionLineID).First(&field).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errProgramCustomFieldNotBelongToProductionLine
				}
				return err
			}
			if !field.Enabled {
				return errProgramCustomFieldDisabled
			}

			value := input.Value
			if field.FieldType == "select" {
				options, err := validateSelectFieldOptions(field.OptionsJSON)
				if err != nil {
					return err
				}
				matched := false
				for _, option := range options {
					if value == option {
						matched = true
						break
					}
				}
				if !matched {
					return errProgramCustomFieldInvalidSelectValue
				}
			}

			newValues = append(newValues, models.ProgramCustomFieldValue{
				ProgramID:                   program.ID,
				ProductionLineCustomFieldID: field.ID,
				Value:                       value,
			})
		}

		if err := tx.Where("program_id = ?", program.ID).Delete(&models.ProgramCustomFieldValue{}).Error; err != nil {
			return err
		}
		if len(newValues) > 0 {
			if err := tx.Create(&newValues).Error; err != nil {
				return err
			}
		}

		savedValues = newValues
		return nil
	})
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
		case errors.Is(err, errProgramCustomFieldFieldIDRequired),
			errors.Is(err, errProgramCustomFieldDuplicateFieldID),
			errors.Is(err, errProgramCustomFieldNotBelongToProductionLine),
			errors.Is(err, errProgramCustomFieldInvalidSelectValue),
			errors.Is(err, errProgramCustomFieldDisabled):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"values": savedValues})
}

var errProgramCustomFieldFieldIDRequired = errors.New("field_id 不能为空")
var errProgramCustomFieldDuplicateFieldID = errors.New("field_id 不能重复")
var errProgramCustomFieldValuesRequired = errors.New("values 不能为空")
var errProgramCustomFieldNotBelongToProductionLine = errors.New("只能保存当前生产线的自定义字段")
var errProgramCustomFieldInvalidSelectValue = errors.New("select 字段的值必须是预设选项之一")
var errProgramCustomFieldDisabled = errors.New("停用的自定义字段不允许写入")
