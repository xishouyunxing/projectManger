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

func replaceProgramCustomFieldValues(tx *gorm.DB, program models.Program, inputs []programCustomFieldValueInput) ([]models.ProgramCustomFieldValue, error) {
	newValues := make([]models.ProgramCustomFieldValue, 0, len(inputs))
	seenFieldIDs := make(map[uint]struct{}, len(inputs))
	for _, input := range inputs {
		if input.FieldID == 0 {
			return nil, errProgramCustomFieldFieldIDRequired
		}
		if _, exists := seenFieldIDs[input.FieldID]; exists {
			return nil, errProgramCustomFieldDuplicateFieldID
		}
		seenFieldIDs[input.FieldID] = struct{}{}

		var field models.ProductionLineCustomField
		if err := tx.Where("id = ? AND production_line_id = ?", input.FieldID, program.ProductionLineID).First(&field).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errProgramCustomFieldNotBelongToProductionLine
			}
			return nil, err
		}
		if !field.Enabled {
			return nil, errProgramCustomFieldDisabled
		}

		value := input.Value
		if field.FieldType == "select" {
			options, err := validateSelectFieldOptions(field.OptionsJSON)
			if err != nil {
				return nil, err
			}
			matched := false
			for _, option := range options {
				if value == option {
					matched = true
					break
				}
			}
			if !matched {
				return nil, errProgramCustomFieldInvalidSelectValue
			}
		}

		newValues = append(newValues, models.ProgramCustomFieldValue{
			ProgramID:                   program.ID,
			ProductionLineCustomFieldID: field.ID,
			Value:                       value,
		})
	}

	if err := tx.Where("program_id = ?", program.ID).Delete(&models.ProgramCustomFieldValue{}).Error; err != nil {
		return nil, err
	}
	if len(newValues) > 0 {
		if err := tx.Create(&newValues).Error; err != nil {
			return nil, err
		}
	}

	return newValues, nil
}

func SaveProgramCustomFieldValues(c *gin.Context) {
	programID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "程序ID格式错误"})
		return
	}

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
	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		_, targetProgramID, _, err := resolveProgramTarget(tx, programID)
		if err != nil {
			return err
		}

		var program models.Program
		if err := tx.First(&program, targetProgramID).Error; err != nil {
			return err
		}
		if !authorizeLineAction(c, program.ProductionLineID, lineActionManage) {
			return errors.New("forbidden")
		}

		savedValues, err = replaceProgramCustomFieldValues(tx, program, req.Values)
		return err
	})
	if txErr != nil {
		switch {
		case errors.Is(txErr, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
		case errors.Is(txErr, errProgramCustomFieldFieldIDRequired),
			errors.Is(txErr, errProgramCustomFieldDuplicateFieldID),
			errors.Is(txErr, errProgramCustomFieldNotBelongToProductionLine),
			errors.Is(txErr, errProgramCustomFieldInvalidSelectValue),
			errors.Is(txErr, errProgramCustomFieldDisabled):
			c.JSON(http.StatusBadRequest, gin.H{"error": txErr.Error()})
		case txErr.Error() == "forbidden":
			return
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
