package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"crane-system/utils"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
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

func buildProgramListResponse(tx *gorm.DB, programs []models.Program, allowedLineIDs map[uint]struct{}) ([]programListItem, error) {
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
			if parent, ok := parentByID[mapping.ParentProgramID]; ok && lineIDAllowed(allowedLineIDs, parent.ProductionLineID) {
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

func applyProgramCustomFieldFilters(query *gorm.DB, rawFilters map[uint]string) (*gorm.DB, error) {
	if len(rawFilters) == 0 {
		return query, nil
	}

	fieldIDs := make([]uint, 0, len(rawFilters))
	for fieldID := range rawFilters {
		fieldIDs = append(fieldIDs, fieldID)
	}

	var fields []models.ProductionLineCustomField
	if err := database.DB.Where("id IN ?", fieldIDs).Find(&fields).Error; err != nil {
		return nil, err
	}

	fieldTypeByID := make(map[uint]string, len(fields))
	for _, field := range fields {
		fieldTypeByID[field.ID] = field.FieldType
	}

	for fieldID, value := range rawFilters {
		trimmedValue := strings.TrimSpace(value)
		if trimmedValue == "" {
			continue
		}

		if fieldTypeByID[fieldID] == "select" {
			query = query.Where(
				"EXISTS (SELECT 1 FROM program_custom_field_values pcfv WHERE pcfv.program_id = programs.id AND pcfv.production_line_custom_field_id = ? AND pcfv.value = ?)",
				fieldID,
				trimmedValue,
			)
			continue
		}

		likeValue := "%" + strings.ToLower(trimmedValue) + "%"
		query = query.Where(
			"EXISTS (SELECT 1 FROM program_custom_field_values pcfv WHERE pcfv.program_id = programs.id AND pcfv.production_line_custom_field_id = ? AND LOWER(COALESCE(pcfv.value, '')) LIKE ?)",
			fieldID,
			likeValue,
		)
	}

	return query, nil
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

	allowedLineIDs, statusCode, message := resolveAuthorizedLineIDs(c, lineActionView)
	if statusCode != 0 {
		c.JSON(statusCode, gin.H{"error": message})
		return
	}

	query := database.DB.Model(&models.Program{})
	if allowedLineIDs != nil {
		lineIDs := make([]uint, 0, len(allowedLineIDs))
		for lineID := range allowedLineIDs {
			lineIDs = append(lineIDs, lineID)
		}
		if len(lineIDs) == 0 {
			if !paged {
				c.JSON(http.StatusOK, []programListItem{})
				return
			}
			c.JSON(http.StatusOK, gin.H{"items": []programListItem{}, "total": 0, "page": page, "page_size": pageSize})
			return
		}
		query = query.Where("production_line_id IN ?", lineIDs)
	}
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

	customFieldFilters := map[uint]string{}
	for key, values := range c.Request.URL.Query() {
		if !strings.HasPrefix(key, "custom_field_") || len(values) == 0 {
			continue
		}

		fieldID, err := strconv.ParseUint(strings.TrimPrefix(key, "custom_field_"), 10, 64)
		if err != nil || fieldID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid custom field filter"})
			return
		}

		if value := strings.TrimSpace(values[0]); value != "" {
			customFieldFilters[uint(fieldID)] = value
		}
	}

	query, err = applyProgramCustomFieldFilters(query, customFieldFilters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
		return
	}

	var total int64
	if paged {
		if err := query.Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
		return
	}

	response, err := buildProgramListResponse(database.DB, programs, allowedLineIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
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
	allowedLineIDs, statusCode, message := resolveAuthorizedLineIDs(c, lineActionView)
	if statusCode != 0 {
		c.JSON(statusCode, gin.H{"error": message})
		return
	}

	query := database.DB.Model(&models.Program{})
	if allowedLineIDs != nil {
		lineIDs := make([]uint, 0, len(allowedLineIDs))
		for lineID := range allowedLineIDs {
			lineIDs = append(lineIDs, lineID)
		}
		if len(lineIDs) == 0 {
			query = query.Where("1 = 0")
		} else {
			query = query.Where("production_line_id IN ?", lineIDs)
		}
	}
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
		return
	}

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheetName := "Programs"
	defaultSheet := f.GetSheetName(f.GetActiveSheetIndex())
	if defaultSheet != sheetName {
		_ = f.SetSheetName(defaultSheet, sheetName)
	}

	headers := []string{"ID", "Program Name", "Program Code", "Production Line", "Vehicle Model", "Status", "Version", "Description", "Created At"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheetName, cell, header); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
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
				c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
				return
			}
		}
	}

	buffer, err := f.WriteToBuffer()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
		return
	}

	utf8FileName := "programs.xlsx"
	asciiFallbackFileName := "programs.xlsx"
	contentDisposition := "attachment; filename=\"" + asciiFallbackFileName + "\"; filename*=UTF-8''" + url.QueryEscape(utf8FileName)

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", contentDisposition)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer.Bytes())
}

func GetProgram(c *gin.Context) {
	programID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??ID????"})
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
		c.JSON(http.StatusNotFound, gin.H{"error": "?????"})
		return
	}
	if !authorizeLineAction(c, program.ProductionLineID, lineActionView) {
		return
	}

	if err := attachProgramMappingInfo(database.DB, &program); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
		return
	}
	if program.MappingInfo != nil {
		parentProgram, _, _, err := resolveProgramTarget(database.DB, program.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
			return
		}
		allowed, statusCode, message := checkLineAction(c, parentProgram.ProductionLineID, lineActionView)
		if !allowed {
			if statusCode != http.StatusForbidden {
				c.JSON(statusCode, gin.H{"error": message})
				return
			}
			program.MappingInfo = nil
		} else {
			applyParentProgramData(&program, parentProgram)
		}
	}

	c.JSON(http.StatusOK, program)
}

type createProgramRequest struct {
	Name              string                          `json:"name" binding:"required"`
	Code              string                          `json:"code" binding:"required"`
	ProductionLineID  uint                            `json:"production_line_id" binding:"required"`
	VehicleModelID    *uint                           `json:"vehicle_model_id"`
	Description       string                          `json:"description"`
	Status            string                          `json:"status"`
	CustomFieldValues *[]programCustomFieldValueInput `json:"custom_field_values"`
}

func validateProgramStatus(status string) error {
	switch status {
	case "in_progress", "completed":
		return nil
	default:
		return errors.New("invalid program status")
	}
}

func validateProgramRelations(tx *gorm.DB, productionLineID uint, vehicleModelID *uint) error {
	if productionLineID == 0 {
		return errors.New("invalid production line")
	}
	var line models.ProductionLine
	if err := tx.Select("id").First(&line, productionLineID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("invalid production line")
		}
		return err
	}

	if vehicleModelID != nil && *vehicleModelID > 0 {
		var vehicleModel models.VehicleModel
		if err := tx.Select("id").First(&vehicleModel, *vehicleModelID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("invalid vehicle model")
			}
			return err
		}
	}
	return nil
}

func CreateProgram(c *gin.Context) {
	var req createProgramRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	req.Status = strings.TrimSpace(req.Status)
	if req.Name == "" || req.Code == "" || req.ProductionLineID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid program payload"})
		return
	}
	if req.Status == "" {
		req.Status = "in_progress"
	}
	if err := validateProgramStatus(req.Status); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateProgramRelations(database.DB, req.ProductionLineID, req.VehicleModelID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !authorizeLineAction(c, req.ProductionLineID, lineActionManage) {
		return
	}

	program := models.Program{
		Name:             req.Name,
		Code:             req.Code,
		ProductionLineID: req.ProductionLineID,
		Description:      req.Description,
		Status:           req.Status,
	}
	if req.VehicleModelID != nil {
		program.VehicleModelID = *req.VehicleModelID
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&program).Error; err != nil {
			return err
		}
		if req.CustomFieldValues != nil {
			_, err := replaceProgramCustomFieldValues(tx, program, *req.CustomFieldValues)
			return err
		}
		return nil
	}); err != nil {
		switch {
		case errors.Is(err, errProgramCustomFieldFieldIDRequired),
			errors.Is(err, errProgramCustomFieldDuplicateFieldID),
			errors.Is(err, errProgramCustomFieldNotBelongToProductionLine),
			errors.Is(err, errProgramCustomFieldInvalidSelectValue),
			errors.Is(err, errProgramCustomFieldDisabled):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
		}
		return
	}

	c.JSON(http.StatusCreated, program)
}

type updateProgramRequest struct {
	Name              *string                         `json:"name"`
	Code              *string                         `json:"code"`
	ProductionLineID  *uint                           `json:"production_line_id"`
	VehicleModelID    optionalVehicleModelIDUpdate    `json:"vehicle_model_id"`
	Description       *string                         `json:"description"`
	Status            *string                         `json:"status"`
	CustomFieldValues *[]programCustomFieldValueInput `json:"custom_field_values"`
}

type optionalVehicleModelIDUpdate struct {
	Set   bool
	Value *uint
}

func (o *optionalVehicleModelIDUpdate) UnmarshalJSON(data []byte) error {
	o.Set = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}

	var value uint
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	o.Value = &value
	return nil
}

func UpdateProgram(c *gin.Context) {
	programID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??ID????"})
		return
	}
	_, targetProgramID, _, err := resolveProgramTarget(database.DB, programID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "?????"})
		return
	}

	var program models.Program
	if err := database.DB.First(&program, targetProgramID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "?????"})
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
	if req.VehicleModelID.Set {
		if req.VehicleModelID.Value == nil {
			updates["vehicle_model_id"] = 0
		} else {
			updates["vehicle_model_id"] = *req.VehicleModelID.Value
		}
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != nil {
		updates["status"] = strings.TrimSpace(*req.Status)
	}

	if len(updates) == 0 && req.CustomFieldValues == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "????????"})
		return
	}

	if req.Status != nil {
		if err := validateProgramStatus(updates["status"].(string)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	nextProductionLineID := originalProductionLineID
	if req.ProductionLineID != nil {
		nextProductionLineID = *req.ProductionLineID
	}
	var nextVehicleModelID *uint
	if req.VehicleModelID.Set && req.VehicleModelID.Value != nil && *req.VehicleModelID.Value > 0 {
		nextVehicleModelID = req.VehicleModelID.Value
	} else if !req.VehicleModelID.Set && program.VehicleModelID > 0 {
		nextVehicleModelID = &program.VehicleModelID
	}
	if err := validateProgramRelations(database.DB, nextProductionLineID, nextVehicleModelID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
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

		updatedProgram := program
		updatedProgram.ProductionLineID = nextProductionLineID
		if req.CustomFieldValues != nil {
			_, err := replaceProgramCustomFieldValues(tx, updatedProgram, *req.CustomFieldValues)
			return err
		}
		if originalProductionLineID != nextProductionLineID {
			if err := tx.Where("program_id = ?", program.ID).Delete(&models.ProgramCustomFieldValue{}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		switch {
		case errors.Is(err, errProgramCustomFieldFieldIDRequired),
			errors.Is(err, errProgramCustomFieldDuplicateFieldID),
			errors.Is(err, errProgramCustomFieldNotBelongToProductionLine),
			errors.Is(err, errProgramCustomFieldInvalidSelectValue),
			errors.Is(err, errProgramCustomFieldDisabled):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
		}
		return
	}

	if err := database.DB.First(&program, targetProgramID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
		return
	}
	c.JSON(http.StatusOK, program)
}

func DeleteProgram(c *gin.Context) {
	programID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??ID????"})
		return
	}

	var filesToDelete []string
	txErr := database.DB.Transaction(func(tx *gorm.DB) error {
		var program models.Program
		if err := tx.First(&program, programID).Error; err != nil {
			return err
		}
		if !authorizeLineAction(c, program.ProductionLineID, lineActionManage) {
			return errors.New("forbidden")
		}

		var files []models.ProgramFile
		if err := tx.Where("program_id = ?", programID).Find(&files).Error; err != nil {
			return err
		}
		for _, file := range files {
			filesToDelete = append(filesToDelete, file.FilePath)
		}

		if err := tx.Where("program_id = ?", programID).Delete(&models.ProgramCustomFieldValue{}).Error; err != nil {
			return err
		}
		if err := tx.Where("program_id = ?", programID).Delete(&models.ProgramVersion{}).Error; err != nil {
			return err
		}
		if err := tx.Where("program_id = ?", programID).Delete(&models.ProgramFile{}).Error; err != nil {
			return err
		}
		if err := tx.Where("parent_program_id = ? OR child_program_id = ?", programID, programID).Delete(&models.ProgramMapping{}).Error; err != nil {
			return err
		}
		if err := tx.Where("source_program_id = ? OR related_program_id = ?", programID, programID).Delete(&models.ProgramRelation{}).Error; err != nil {
			return err
		}
		return tx.Delete(&program).Error
	})
	if txErr != nil {
		if txErr.Error() == "forbidden" {
			return
		}
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "?????"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
		return
	}

	uploadDir := utils.UploadDir()
	for _, filePath := range filesToDelete {
		fullPath := filepath.Join(uploadDir, filePath)
		if utils.IsSafePath(uploadDir, fullPath) {
			_ = utils.DeleteFile(fullPath)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "????"})
}

func GetProgramsByVehicle(c *gin.Context) {
	vehicleID, err := parseUintParam(c.Param("vehicle_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??ID????"})
		return
	}

	allowedLineIDs, statusCode, message := resolveAuthorizedLineIDs(c, lineActionView)
	if statusCode != 0 {
		c.JSON(statusCode, gin.H{"error": message})
		return
	}

	query := database.DB.
		Preload("ProductionLine").
		Preload("ProductionLine.Process").
		Where("vehicle_model_id = ?", vehicleID)
	if allowedLineIDs != nil {
		lineIDs := make([]uint, 0, len(allowedLineIDs))
		for lineID := range allowedLineIDs {
			lineIDs = append(lineIDs, lineID)
		}
		if len(lineIDs) == 0 {
			c.JSON(http.StatusOK, []models.Program{})
			return
		}
		query = query.Where("production_line_id IN ?", lineIDs)
	}

	var programs []models.Program
	if err := query.Find(&programs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
		return
	}

	c.JSON(http.StatusOK, programs)
}

func GetProgramRelations(c *gin.Context) {
	programID, err := parseUintParam(c.Param("program_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??ID????"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
		return
	}

	visibleRelations := make([]models.ProgramRelation, 0, len(relations))
	for _, relation := range relations {
		visible, statusCode, message := programRelationVisible(c, relation)
		if statusCode != 0 {
			c.JSON(statusCode, gin.H{"error": message})
			return
		}
		if visible {
			visibleRelations = append(visibleRelations, relation)
		}
	}

	c.JSON(http.StatusOK, visibleRelations)
}

func programRelationVisible(c *gin.Context, relation models.ProgramRelation) (bool, int, string) {
	sourceVisible, statusCode, message := relationProgramVisible(c, relation.SourceProgram)
	if statusCode != 0 || !sourceVisible {
		return sourceVisible, statusCode, message
	}
	return relationProgramVisible(c, relation.RelatedProgram)
}

func relationProgramVisible(c *gin.Context, program models.Program) (bool, int, string) {
	if program.ID == 0 {
		return false, http.StatusInternalServerError, "????"
	}
	allowed, statusCode, message := checkLineAction(c, program.ProductionLineID, lineActionView)
	if allowed {
		return true, 0, ""
	}
	if statusCode == http.StatusForbidden {
		return false, 0, ""
	}
	return false, statusCode, message
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
		return
	}

	c.JSON(http.StatusCreated, relation)
}

func DeleteRelation(c *gin.Context) {
	relationID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??ID????"})
		return
	}

	var relation models.ProgramRelation
	if err := database.DB.First(&relation, relationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "?????"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "????"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "????"})
}
