package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type productionLineCustomFieldRequest struct {
	Name        *string `json:"name"`
	FieldType   *string `json:"field_type"`
	OptionsJSON *string `json:"options_json"`
	SortOrder   *int    `json:"sort_order"`
	Enabled     *bool   `json:"enabled"`
}

func GetProductionLineCustomFields(c *gin.Context) {
	lineID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "生产线ID格式错误"})
		return
	}
	if _, err := findProductionLine(lineID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "生产线不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		}
		return
	}

	var fields []models.ProductionLineCustomField
	if err := database.DB.Where("production_line_id = ?", lineID).Order("sort_order asc, id asc").Find(&fields).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, fields)
}

func CreateProductionLineCustomField(c *gin.Context) {
	lineID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "生产线ID格式错误"})
		return
	}

	line, err := findProductionLine(lineID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "生产线不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		}
		return
	}

	var req productionLineCustomFieldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	field, err := buildProductionLineCustomFieldForCreate(line.ID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateProductionLineCustomFieldNameUnique(field.ProductionLineID, field.Name, 0); err != nil {
		if errors.Is(err, errDuplicateProductionLineCustomFieldName) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		}
		return
	}

	if err := database.DB.Create(&field).Error; err != nil {
		if isDuplicateProductionLineCustomFieldError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "同一生产线下字段名称不能重复"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusCreated, field)
}

func UpdateProductionLineCustomField(c *gin.Context) {
	lineID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "生产线ID格式错误"})
		return
	}
	fieldID, err := parseUintParam(c.Param("fieldId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "字段ID格式错误"})
		return
	}

	if _, err := findProductionLine(lineID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "生产线不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		}
		return
	}

	field, err := findProductionLineCustomField(lineID, fieldID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "自定义字段不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		}
		return
	}

	var req productionLineCustomFieldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedField, err := buildProductionLineCustomFieldForUpdate(field, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateProductionLineCustomFieldNameUnique(updatedField.ProductionLineID, updatedField.Name, updatedField.ID); err != nil {
		if errors.Is(err, errDuplicateProductionLineCustomFieldName) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		}
		return
	}

	field.Name = updatedField.Name
	field.FieldType = updatedField.FieldType
	field.OptionsJSON = updatedField.OptionsJSON
	field.SortOrder = updatedField.SortOrder
	field.Enabled = updatedField.Enabled

	if err := database.DB.Save(&field).Error; err != nil {
		if isDuplicateProductionLineCustomFieldError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "同一生产线下字段名称不能重复"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, field)
}

func DeleteProductionLineCustomField(c *gin.Context) {
	lineID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "生产线ID格式错误"})
		return
	}
	fieldID, err := parseUintParam(c.Param("fieldId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "字段ID格式错误"})
		return
	}

	if _, err := findProductionLine(lineID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "生产线不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		}
		return
	}

	field, err := findProductionLineCustomField(lineID, fieldID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "自定义字段不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		}
		return
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		var dependentValues int64
		if err := tx.Model(&models.ProgramCustomFieldValue{}).Where("production_line_custom_field_id = ?", field.ID).Count(&dependentValues).Error; err != nil {
			return err
		}
		if dependentValues > 0 {
			return errProductionLineCustomFieldInUse
		}
		return tx.Delete(&field).Error
	}); err != nil {
		if errors.Is(err, errProductionLineCustomFieldInUse) {
			c.JSON(http.StatusConflict, gin.H{"error": "该自定义字段已被程序使用，无法删除"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func buildProductionLineCustomFieldForCreate(lineID uint, req productionLineCustomFieldRequest) (models.ProductionLineCustomField, error) {
	field := models.ProductionLineCustomField{
		ProductionLineID: lineID,
		Enabled:          true,
	}
	if req.Name != nil {
		field.Name = strings.TrimSpace(*req.Name)
	}
	if req.FieldType != nil {
		field.FieldType = strings.TrimSpace(*req.FieldType)
	}
	if req.OptionsJSON != nil {
		field.OptionsJSON = strings.TrimSpace(*req.OptionsJSON)
	}
	if req.SortOrder != nil {
		field.SortOrder = *req.SortOrder
	}
	if req.Enabled != nil {
		field.Enabled = *req.Enabled
	}

	return validateProductionLineCustomField(field)
}

func buildProductionLineCustomFieldForUpdate(existing models.ProductionLineCustomField, req productionLineCustomFieldRequest) (models.ProductionLineCustomField, error) {
	field := existing
	if req.Name != nil {
		field.Name = strings.TrimSpace(*req.Name)
	}
	if req.FieldType != nil {
		field.FieldType = strings.TrimSpace(*req.FieldType)
	}
	if req.OptionsJSON != nil {
		field.OptionsJSON = strings.TrimSpace(*req.OptionsJSON)
	}
	if req.SortOrder != nil {
		field.SortOrder = *req.SortOrder
	}
	if req.Enabled != nil {
		field.Enabled = *req.Enabled
	}

	return validateProductionLineCustomField(field)
}

func validateProductionLineCustomField(field models.ProductionLineCustomField) (models.ProductionLineCustomField, error) {

	if field.Name == "" {
		return models.ProductionLineCustomField{}, errors.New("字段名称不能为空")
	}

	if field.FieldType != "text" && field.FieldType != "select" {
		return models.ProductionLineCustomField{}, errors.New("field_type 仅支持 text 或 select")
	}

	if field.FieldType == "select" {
		options, err := validateSelectFieldOptions(field.OptionsJSON)
		if err != nil {
			return models.ProductionLineCustomField{}, err
		}
		normalized, err := json.Marshal(options)
		if err != nil {
			return models.ProductionLineCustomField{}, errors.New("options_json 无效")
		}
		field.OptionsJSON = string(normalized)
	} else {
		field.OptionsJSON = ""
	}

	return field, nil
}

func validateSelectFieldOptions(optionsJSON string) ([]string, error) {
	if strings.TrimSpace(optionsJSON) == "" {
		return nil, errors.New("select 类型必须提供至少一个有效选项")
	}

	var rawOptions []string
	if err := json.Unmarshal([]byte(optionsJSON), &rawOptions); err != nil {
		return nil, errors.New("options_json 无效")
	}

	options := make([]string, 0, len(rawOptions))
	for _, option := range rawOptions {
		option = strings.TrimSpace(option)
		if option != "" {
			options = append(options, option)
		}
	}

	if len(options) == 0 {
		return nil, errors.New("select 类型必须提供至少一个有效选项")
	}

	return options, nil
}

func findProductionLine(lineID uint) (models.ProductionLine, error) {
	var line models.ProductionLine
	if err := database.DB.First(&line, lineID).Error; err != nil {
		return models.ProductionLine{}, err
	}
	return line, nil
}

func findProductionLineCustomField(lineID uint, fieldID uint) (models.ProductionLineCustomField, error) {
	var field models.ProductionLineCustomField
	if err := database.DB.Where("id = ? AND production_line_id = ?", fieldID, lineID).First(&field).Error; err != nil {
		return models.ProductionLineCustomField{}, err
	}
	return field, nil
}

func isDuplicateProductionLineCustomFieldError(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(strings.ToLower(err.Error()), "unique")
}

var errProductionLineCustomFieldInUse = errors.New("production line custom field in use")
var errDuplicateProductionLineCustomFieldName = errors.New("同一生产线下字段名称不能重复")

func validateProductionLineCustomFieldNameUnique(lineID uint, name string, excludeID uint) error {
	var count int64
	query := database.DB.Model(&models.ProductionLineCustomField{}).Where("production_line_id = ? AND name = ?", lineID, name)
	if excludeID != 0 {
		query = query.Where("id <> ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errDuplicateProductionLineCustomFieldName
	}
	return nil
}
