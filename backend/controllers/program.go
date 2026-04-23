package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

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

type programCountRow struct {
	ProgramID uint  `gorm:"column:program_id"`
	Total     int64 `gorm:"column:total"`
}

func buildProgramCountMap(tx *gorm.DB, model any, programIDs []uint) (map[uint]int64, error) {
	counts := make(map[uint]int64, len(programIDs))
	if len(programIDs) == 0 {
		return counts, nil
	}

	var rows []programCountRow
	if err := tx.Model(model).
		Select("program_id, COUNT(*) AS total").
		Where("program_id IN ?", programIDs).
		Group("program_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		counts[row.ProgramID] = row.Total
	}
	return counts, nil
}

func buildProgramVersionCountMap(tx *gorm.DB, programIDs []uint) (map[uint]int64, error) {
	return buildProgramCountMap(tx, &models.ProgramVersion{}, programIDs)
}

func buildProgramFileCountMap(tx *gorm.DB, programIDs []uint) (map[uint]int64, error) {
	return buildProgramCountMap(tx, &models.ProgramFile{}, programIDs)
}

func buildProgramParentDataMap(tx *gorm.DB, programIDs []uint) (map[uint]models.ProgramMapping, map[uint]models.Program, error) {
	mappingByChildID := make(map[uint]models.ProgramMapping)
	parentByID := make(map[uint]models.Program)
	if len(programIDs) == 0 {
		return mappingByChildID, parentByID, nil
	}

	var mappings []models.ProgramMapping
	if err := tx.Where("child_program_id IN ?", programIDs).Find(&mappings).Error; err != nil {
		if strings.Contains(err.Error(), "no such table") {
			return mappingByChildID, parentByID, nil
		}
		return nil, nil, err
	}
	parentIDs := make([]uint, 0, len(mappings))
	for _, mapping := range mappings {
		mappingByChildID[mapping.ChildProgramID] = mapping
		parentIDs = append(parentIDs, mapping.ParentProgramID)
	}
	if len(parentIDs) == 0 {
		return mappingByChildID, parentByID, nil
	}

	var parents []models.Program
	if err := tx.Preload("ProductionLine").Preload("VehicleModel").Where("id IN ?", parentIDs).Find(&parents).Error; err != nil {
		return nil, nil, err
	}
	for _, parent := range parents {
		parentByID[parent.ID] = parent
	}

	return mappingByChildID, parentByID, nil
}

func buildProgramListResponse(tx *gorm.DB, programs []models.Program) ([]programListItem, error) {
	response := make([]programListItem, 0, len(programs))
	if len(programs) == 0 {
		return response, nil
	}

	programIDs := make([]uint, 0, len(programs))
	for _, program := range programs {
		programIDs = append(programIDs, program.ID)
	}

	versionCounts, err := buildProgramVersionCountMap(tx, programIDs)
	if err != nil {
		return nil, err
	}
	fileCounts, err := buildProgramFileCountMap(tx, programIDs)
	if err != nil {
		return nil, err
	}
	mappingByChildID, parentByID, err := buildProgramParentDataMap(tx, programIDs)
	if err != nil {
		return nil, err
	}

	for _, program := range programs {
		effectiveProgram := program
		effectiveProgram.OwnVersionCount = versionCounts[program.ID]
		effectiveProgram.OwnFileCount = fileCounts[program.ID]

		if mapping, ok := mappingByChildID[program.ID]; ok {
			if parent, ok := parentByID[mapping.ParentProgramID]; ok {
				effectiveProgram.MappingInfo = &models.ProgramMappingInfo{
					MappingID:         mapping.ID,
					ParentProgramID:   parent.ID,
					ParentProgramName: parent.Name,
					ParentProgramCode: parent.Code,
				}
				applyParentProgramData(&effectiveProgram, parent)
			} else {
				effectiveProgram.MappingInfo = &models.ProgramMappingInfo{
					MappingID:       mapping.ID,
					ParentProgramID: mapping.ParentProgramID,
				}
			}
		}

		item := programListItem{Program: effectiveProgram}
		item.CustomFieldValues = summarizeEnabledProgramCustomFieldValues(effectiveProgram.CustomFieldValues)
		response = append(response, item)
	}

	return response, nil
}

func GetPrograms(c *gin.Context) {
	pageQuery := c.Query("page")
	pageSizeQuery := c.Query("page_size")
	paged := pageQuery != "" || pageSizeQuery != ""

	page, err := parsePositiveIntQuery(pageQuery, 1, 0, "page")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pageSize, err := parsePositiveIntQuery(pageSizeQuery, 20, 200, "page_size")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := database.DB.Model(&models.Program{})
	if lineID := c.Query("production_line_id"); lineID != "" {
		query = query.Where("production_line_id = ?", lineID)
	}
	if vehicleID := c.Query("vehicle_model_id"); vehicleID != "" {
		query = query.Where("vehicle_model_id = ?", vehicleID)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		likeKeyword := "%" + strings.ToLower(keyword) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(code) LIKE ?", likeKeyword, likeKeyword)
	}

	var total int64
	if paged {
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
			return
		}
	}

	query = query.
		Preload("ProductionLine").
		Preload("VehicleModel").
		Preload("CustomFieldValues").
		Preload("CustomFieldValues.ProductionLineCustomField").
		Order("id DESC")
	if paged {
		query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	}

	var programs []models.Program
	if err := query.Find(&programs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	response, err := buildProgramListResponse(database.DB, programs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	if !paged {
		c.JSON(http.StatusOK, response)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":     response,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
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

func ExportProgramsExcel(c *gin.Context) {
	query := database.DB.Model(&models.Program{})
	if lineID := c.Query("production_line_id"); lineID != "" {
		query = query.Where("production_line_id = ?", lineID)
	}
	if vehicleID := c.Query("vehicle_model_id"); vehicleID != "" {
		query = query.Where("vehicle_model_id = ?", vehicleID)
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}

	var programs []models.Program
	if err := query.
		Preload("ProductionLine").
		Preload("VehicleModel").
		Order("id DESC").
		Find(&programs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "导出失败"})
		return
	}

	for _, program := range programs {
		if !authorizeLineAction(c, program.ProductionLineID, lineActionView) {
			return
		}
	}

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheetName := "程序列表"
	defaultSheet := f.GetSheetName(f.GetActiveSheetIndex())
	if defaultSheet != sheetName {
		_ = f.SetSheetName(defaultSheet, sheetName)
	}

	headers := []string{"ID", "程序名称", "程序编码", "生产线", "车型", "状态", "版本", "描述", "创建时间"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheetName, cell, header); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "导出失败"})
			return
		}
	}

	for rowIdx, program := range programs {
		row := rowIdx + 2
		createdAt := ""
		if !program.CreatedAt.IsZero() {
			createdAt = program.CreatedAt.Format(time.DateTime)
		}
		rowValues := []any{
			program.ID,
			program.Name,
			program.Code,
			program.ProductionLine.Name,
			program.VehicleModel.Name,
			program.Status,
			program.Version,
			program.Description,
			createdAt,
		}
		for colIdx, value := range rowValues {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, row)
			if err := f.SetCellValue(sheetName, cell, value); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "导出失败"})
				return
			}
		}
	}

	buffer, err := f.WriteToBuffer()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "导出失败"})
		return
	}

	utf8FileName := "程序列表.xlsx"
	asciiFallbackFileName := "programs.xlsx"
	contentDisposition := "attachment; filename=\"" + asciiFallbackFileName + "\"; filename*=UTF-8''" + url.QueryEscape(utf8FileName)

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", contentDisposition)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer.Bytes())
}

func GetProgram(c *gin.Context) {
	programID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "程序ID格式错误"})
		return
	}

	var program models.Program
	if err := database.DB.
		Preload("ProductionLine").
		Preload("VehicleModel").
		Preload("Files").
		Preload("Versions").
		Preload("CustomFieldValues").
		First(&program, programID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
		return
	}
	if !authorizeLineAction(c, program.ProductionLineID, lineActionView) {
		return
	}

	if err := attachProgramMappingInfo(database.DB, &program); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	if program.MappingInfo != nil {
		parentProgram, _, _, err := resolveProgramTarget(database.DB, program.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
			return
		}
		applyParentProgramData(&program, parentProgram)
	}

	c.JSON(http.StatusOK, program)
}

func CreateProgram(c *gin.Context) {
	var program models.Program
	if err := c.ShouldBindJSON(&program); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !authorizeLineAction(c, program.ProductionLineID, lineActionManage) {
		return
	}

	if err := database.DB.Create(&program).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusCreated, program)
}

type updateProgramRequest struct {
	Name             *string `json:"name"`
	Code             *string `json:"code"`
	ProductionLineID *uint   `json:"production_line_id"`
	VehicleModelID   *uint   `json:"vehicle_model_id"`
	Description      *string `json:"description"`
	Status           *string `json:"status"`
}

func UpdateProgram(c *gin.Context) {
	programID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "程序ID格式错误"})
		return
	}
	_, targetProgramID, _, err := resolveProgramTarget(database.DB, programID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
		return
	}

	var program models.Program
	if err := database.DB.First(&program, targetProgramID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
		return
	}
	originalProductionLineID := program.ProductionLineID
	if !authorizeLineAction(c, originalProductionLineID, lineActionManage) {
		return
	}

	var req updateProgramRequest
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
	if req.ProductionLineID != nil {
		updates["production_line_id"] = *req.ProductionLineID
	}
	if req.VehicleModelID != nil {
		updates["vehicle_model_id"] = *req.VehicleModelID
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

	nextProductionLineID := originalProductionLineID
	if req.ProductionLineID != nil {
		nextProductionLineID = *req.ProductionLineID
	}
	if nextProductionLineID != originalProductionLineID {
		if !authorizeLineAction(c, nextProductionLineID, lineActionManage) {
			return
		}
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&program).Updates(updates).Error; err != nil {
			return err
		}
		if originalProductionLineID != nextProductionLineID {
			if err := tx.Where("program_id = ?", program.ID).Delete(&models.ProgramCustomFieldValue{}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	if err := database.DB.First(&program, targetProgramID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, program)
}

func DeleteProgram(c *gin.Context) {
	programID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "程序ID格式错误"})
		return
	}

	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		var program models.Program
		if err := tx.First(&program, programID).Error; err != nil {
			return err
		}
		if !authorizeLineAction(c, program.ProductionLineID, lineActionManage) {
			return errors.New("forbidden")
		}
		if err := tx.Where("program_id = ?", programID).Delete(&models.ProgramCustomFieldValue{}).Error; err != nil {
			return err
		}
		return tx.Delete(&program).Error
	})
	if txErr != nil {
		if txErr.Error() == "forbidden" {
			return
		}
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "程序不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func GetProgramsByVehicle(c *gin.Context) {
	vehicleID, err := parseUintParam(c.Param("vehicle_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "车型ID格式错误"})
		return
	}

	var programs []models.Program
	if err := database.DB.
		Preload("ProductionLine").
		Preload("ProductionLine.Process").
		Where("vehicle_model_id = ?", vehicleID).
		Find(&programs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	filtered := make([]models.Program, 0, len(programs))
	for _, program := range programs {
		allowed, statusCode, message := checkLineAction(c, program.ProductionLineID, lineActionView)
		if !allowed {
			if statusCode == http.StatusForbidden {
				continue
			}
			c.JSON(statusCode, gin.H{"error": message})
			return
		}
		filtered = append(filtered, program)
	}

	c.JSON(http.StatusOK, filtered)
}

func GetProgramRelations(c *gin.Context) {
	programID, err := parseUintParam(c.Param("program_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "程序ID格式错误"})
		return
	}
	if !authorizeProgramAction(c, database.DB, programID, lineActionView) {
		return
	}

	var relations []models.ProgramRelation
	if err := database.DB.
		Preload("SourceProgram").
		Preload("RelatedProgram").
		Where("source_program_id = ? OR related_program_id = ?", programID, programID).
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
	if !authorizeProgramAction(c, database.DB, relation.SourceProgramID, lineActionManage) {
		return
	}
	if !authorizeProgramAction(c, database.DB, relation.RelatedProgramID, lineActionManage) {
		return
	}

	if err := database.DB.Create(&relation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusCreated, relation)
}

func DeleteRelation(c *gin.Context) {
	relationID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "关联ID格式错误"})
		return
	}

	var relation models.ProgramRelation
	if err := database.DB.First(&relation, relationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "关联不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	if !authorizeProgramAction(c, database.DB, relation.SourceProgramID, lineActionManage) {
		return
	}
	if !authorizeProgramAction(c, database.DB, relation.RelatedProgramID, lineActionManage) {
		return
	}

	result := database.DB.Delete(&models.ProgramRelation{}, relationID)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
