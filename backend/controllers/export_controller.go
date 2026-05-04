package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// ──────────────────────────────────────────────────────────────
// 列元数据
// ──────────────────────────────────────────────────────────────

type exportColumnDef struct {
	Key              string `json:"key"`
	Label            string `json:"label"`
	Group            string `json:"group"`
	FieldType        string `json:"field_type,omitempty"`
	ProductionLineID uint   `json:"production_line_id,omitempty"`
}

func builtinExportColumns() []exportColumnDef {
	return []exportColumnDef{
		{Key: "name", Label: "程序名称", Group: "基本信息"},
		{Key: "code", Label: "程序编号", Group: "基本信息"},
		{Key: "production_line", Label: "生产线", Group: "归属"},
		{Key: "vehicle_model", Label: "车型", Group: "归属"},
		{Key: "status", Label: "状态", Group: "基本信息"},
		{Key: "version", Label: "当前版本", Group: "版本"},
		{Key: "description", Label: "描述", Group: "基本信息"},
		{Key: "created_at", Label: "创建时间", Group: "时间"},
		{Key: "file_count", Label: "文件数", Group: "统计"},
		{Key: "version_count", Label: "版本数", Group: "统计"},
	}
}

// GetExportColumns 返回当前用户可见的所有可用列定义。
func GetExportColumns(c *gin.Context) {
	allowedLineIDs, statusCode, msg := resolveAuthorizedLineIDs(c, lineActionView)
	if statusCode != 0 {
		c.JSON(statusCode, gin.H{"error": msg})
		return
	}

	// 查询自定义字段
	var customFields []models.ProductionLineCustomField
	query := database.DB.Where("enabled = ?", true)
	if allowedLineIDs != nil {
		lineIDs := make([]uint, 0, len(allowedLineIDs))
		for id := range allowedLineIDs {
			lineIDs = append(lineIDs, id)
		}
		if len(lineIDs) == 0 {
			customFields = nil
		} else {
			query = query.Where("production_line_id IN ?", lineIDs)
		}
	}
	if err := query.Order("sort_order ASC, id ASC").Find(&customFields).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询自定义字段失败"})
		return
	}

	cfColumns := make([]exportColumnDef, 0, len(customFields))
	for _, cf := range customFields {
		cfColumns = append(cfColumns, exportColumnDef{
			Key:              fmt.Sprintf("cf_%d", cf.ID),
			Label:            cf.Name,
			Group:            "自定义字段",
			FieldType:        cf.FieldType,
			ProductionLineID: cf.ProductionLineID,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"builtin_fields": builtinExportColumns(),
		"custom_fields":  cfColumns,
	})
}

// ──────────────────────────────────────────────────────────────
// 共享查询逻辑
// ──────────────────────────────────────────────────────────────

// parseColumnKeys 解析逗号分隔的列 key 列表；为空时返回默认列。
func parseColumnKeys(c *gin.Context) []string {
	raw := strings.TrimSpace(c.Query("columns"))
	if raw == "" {
		return []string{
			"name", "code", "production_line", "vehicle_model",
			"status", "version", "description", "created_at",
		}
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// parseLineIDs 解析 line_ids 参数（逗号分隔）。
func parseLineIDs(c *gin.Context) []uint {
	raw := strings.TrimSpace(c.Query("line_ids"))
	if raw == "" {
		return nil
	}
	var ids []uint
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		v, err := strconv.ParseUint(s, 10, 64)
		if err == nil && v > 0 {
			ids = append(ids, uint(v))
		}
	}
	return ids
}

// exportQueryPrograms 构建带权限 + 筛选的程序查询。
func exportQueryPrograms(c *gin.Context, extraLineIDs []uint) (*gorm.DB, []uint, int, string) {
	allowedLineIDs, statusCode, msg := resolveAuthorizedLineIDs(c, lineActionView)
	if statusCode != 0 {
		return nil, nil, statusCode, msg
	}

	query := database.DB.Model(&models.Program{})

	// 合并权限产线 + 用户指定产线
	if allowedLineIDs != nil {
		lineIDs := make([]uint, 0, len(allowedLineIDs))
		for id := range allowedLineIDs {
			lineIDs = append(lineIDs, id)
		}
		if len(lineIDs) == 0 {
			query = query.Where("1 = 0")
		} else if len(extraLineIDs) > 0 {
			// 取交集
			allowed := make(map[uint]struct{}, len(lineIDs))
			for _, id := range lineIDs {
				allowed[id] = struct{}{}
			}
			var intersection []uint
			for _, id := range extraLineIDs {
				if _, ok := allowed[id]; ok {
					intersection = append(intersection, id)
				}
			}
			if len(intersection) == 0 {
				query = query.Where("1 = 0")
			} else {
				query = query.Where("production_line_id IN ?", intersection)
			}
		} else {
			query = query.Where("production_line_id IN ?", lineIDs)
		}
	} else if len(extraLineIDs) > 0 {
		query = query.Where("production_line_id IN ?", extraLineIDs)
	}

	filterQuery, filterErr := applyProgramRequestFilters(c, query)
	if filterErr != nil {
		return nil, nil, filterErr.Status, filterErr.Message
	}

	// 返回合并后的产线 ID 列表（用于后续列元数据过滤）
	var mergedLineIDs []uint
	if allowedLineIDs != nil {
		for id := range allowedLineIDs {
			mergedLineIDs = append(mergedLineIDs, id)
		}
	} else {
		mergedLineIDs = extraLineIDs
	}

	return filterQuery, mergedLineIDs, 0, ""
}

// ──────────────────────────────────────────────────────────────
// 预览接口
// ──────────────────────────────────────────────────────────────

// ExportPreview 返回预览数据（JSON），与导出共享查询逻辑。
func ExportPreview(c *gin.Context) {
	extraLineIDs := parseLineIDs(c)
	query, _, statusCode, msg := exportQueryPrograms(c, extraLineIDs)
	if statusCode != 0 {
		c.JSON(statusCode, gin.H{"error": msg})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}

	columnKeys := parseColumnKeys(c)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	var programs []models.Program
	if err := query.
		Preload("ProductionLine").
		Preload("VehicleModel").
		Preload("CustomFieldValues").
		Preload("CustomFieldValues.ProductionLineCustomField").
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&programs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	// 加载版本数和文件数
	programIDs := make([]uint, 0, len(programs))
	for _, p := range programs {
		programIDs = append(programIDs, p.ID)
	}
	versionCounts, _ := buildProgramVersionCountMap(database.DB, programIDs)
	fileCounts, _ := buildProgramFileCountMap(database.DB, programIDs)
	for i := range programs {
		programs[i].OwnVersionCount = versionCounts[programs[i].ID]
		programs[i].OwnFileCount = fileCounts[programs[i].ID]
	}

	// 查询自定义字段定义用于表头
	cfDefMap := loadCustomFieldDefMap()

	// 构建列头信息
	columnHeaders := buildColumnHeaders(columnKeys, cfDefMap)

	// 构建行数据
	items := make([]map[string]any, 0, len(programs))
	for _, p := range programs {
		row := buildExportRow(p, columnKeys)
		items = append(items, row)
	}

	c.JSON(http.StatusOK, gin.H{
		"items":   items,
		"total":   total,
		"page":    page,
		"page_size": pageSize,
		"columns": columnHeaders,
	})
}

// ──────────────────────────────────────────────────────────────
// 统计接口（避免前端拉取全量数据）
// ──────────────────────────────────────────────────────────────

// ExportStats 返回程序统计数据，用于前端统计卡片。
func ExportStats(c *gin.Context) {
	extraLineIDs := parseLineIDs(c)
	query, _, statusCode, msg := exportQueryPrograms(c, extraLineIDs)
	if statusCode != 0 {
		c.JSON(statusCode, gin.H{"error": msg})
		return
	}

	type lineModelStatus struct {
		LineID   uint   `gorm:"column:line_id"`
		ModelID  uint   `gorm:"column:model_id"`
		Status   string `gorm:"column:status"`
		Cnt      int    `gorm:"column:cnt"`
	}

	var rows []lineModelStatus
	if err := query.
		Select("production_line_id AS line_id, vehicle_model_id AS model_id, status, COUNT(*) AS cnt").
		Group("production_line_id, vehicle_model_id, status").
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "统计查询失败"})
		return
	}

	totalPrograms := 0
	completedPrograms := 0
	lineTotal := make(map[uint]int)
	lineCompleted := make(map[uint]int)
	modelTotal := make(map[uint]int)
	modelCompleted := make(map[uint]int)
	cellSet := make(map[string]bool)

	for _, r := range rows {
		totalPrograms += r.Cnt
		lineTotal[r.LineID] += r.Cnt
		modelTotal[r.ModelID] += r.Cnt
		cellSet[fmt.Sprintf("%d:%d", r.LineID, r.ModelID)] = true
		if r.Status == "completed" {
			completedPrograms += r.Cnt
			lineCompleted[r.LineID] += r.Cnt
			modelCompleted[r.ModelID] += r.Cnt
		}
	}

	// 加载产线名称
	var allLines []models.ProductionLine
	{
		lineQuery := database.DB.Model(&models.ProductionLine{})
		if allowedLineIDs, statusCode, _ := resolveAuthorizedLineIDs(c, lineActionView); statusCode == 0 && allowedLineIDs != nil {
			ids := make([]uint, 0, len(allowedLineIDs))
			for id := range allowedLineIDs {
				ids = append(ids, id)
			}
			if len(ids) > 0 {
				if len(extraLineIDs) > 0 {
					allowed := make(map[uint]struct{}, len(ids))
					for _, id := range ids {
						allowed[id] = struct{}{}
					}
					var intersection []uint
					for _, id := range extraLineIDs {
						if _, ok := allowed[id]; ok {
							intersection = append(intersection, id)
						}
					}
					if len(intersection) > 0 {
						lineQuery = lineQuery.Where("id IN ?", intersection)
					} else {
						lineQuery = lineQuery.Where("1 = 0")
					}
				} else {
					lineQuery = lineQuery.Where("id IN ?", ids)
				}
			} else {
				lineQuery = lineQuery.Where("1 = 0")
			}
		} else if len(extraLineIDs) > 0 {
			lineQuery = lineQuery.Where("id IN ?", extraLineIDs)
		}
		lineQuery.Find(&allLines)
	}

	// 加载车型名称
	var allModels []models.VehicleModel
	database.DB.Find(&allModels)

	type rateItem struct {
		Name  string `json:"name"`
		Rate  int    `json:"rate"`
		Total int    `json:"total"`
	}

	lineRates := make([]rateItem, 0, len(allLines))
	for _, l := range allLines {
		total := lineTotal[l.ID]
		completed := lineCompleted[l.ID]
		rate := 0
		if total > 0 {
			rate = int(float64(completed) / float64(total) * 100)
		}
		lineRates = append(lineRates, rateItem{Name: l.Name, Rate: rate, Total: total})
	}

	modelRates := make([]rateItem, 0, len(allModels))
	for _, m := range allModels {
		total := modelTotal[m.ID]
		completed := modelCompleted[m.ID]
		rate := 0
		if total > 0 {
			rate = int(float64(completed) / float64(total) * 100)
		}
		modelRates = append(modelRates, rateItem{Name: m.Name, Rate: rate, Total: total})
	}

	totalPairs := len(allLines) * len(allModels)
	overallRate := 0
	if totalPairs > 0 {
		overallRate = int(float64(len(cellSet)) / float64(totalPairs) * 100)
	}

	c.JSON(http.StatusOK, gin.H{
		"total_programs":      totalPrograms,
		"completed_programs":  completedPrograms,
		"in_progress_programs": totalPrograms - completedPrograms,
		"total_lines":         len(allLines),
		"total_models":        len(allModels),
		"overall_rate":        overallRate,
		"line_rates":          lineRates,
		"model_rates":         modelRates,
	})
}

// loadCustomFieldDefMap 查询所有启用的自定义字段定义，返回 ID→定义 映射。
func loadCustomFieldDefMap() map[uint]models.ProductionLineCustomField {
	m := make(map[uint]models.ProductionLineCustomField)
	var all []models.ProductionLineCustomField
	if err := database.DB.Where("enabled = ?", true).Find(&all).Error; err == nil {
		for _, cf := range all {
			m[cf.ID] = cf
		}
	}
	return m
}

func buildColumnHeaders(keys []string, cfDefMap map[uint]models.ProductionLineCustomField) []map[string]string {
	headers := make([]map[string]string, 0, len(keys))
	builtinMap := make(map[string]string, len(builtinExportColumns()))
	for _, col := range builtinExportColumns() {
		builtinMap[col.Key] = col.Label
	}
	for _, key := range keys {
		label := builtinMap[key]
		if label == "" && strings.HasPrefix(key, "cf_") {
			idStr := strings.TrimPrefix(key, "cf_")
			if id, err := strconv.ParseUint(idStr, 10, 64); err == nil {
				if cf, ok := cfDefMap[uint(id)]; ok {
					label = cf.Name
				}
			}
			if label == "" {
				label = key
			}
		}
		headers = append(headers, map[string]string{"key": key, "label": label})
	}
	return headers
}

func statusLabel(status string) string {
	switch status {
	case "completed":
		return "已完成"
	case "in_progress":
		return "进行中"
	default:
		return status
	}
}

// buildExportRow 按列 key 组装一行数据。
func buildExportRow(p models.Program, keys []string) map[string]any {
	// 预建自定义字段值映射
	cfMap := make(map[uint]string, len(p.CustomFieldValues))
	for _, v := range p.CustomFieldValues {
		if v.ProductionLineCustomField.ID > 0 && v.ProductionLineCustomField.Enabled {
			cfMap[v.ProductionLineCustomFieldID] = v.Value
		}
	}

	row := make(map[string]any, len(keys))
	for _, key := range keys {
		switch key {
		case "name":
			row[key] = p.Name
		case "code":
			row[key] = p.Code
		case "production_line":
			row[key] = p.ProductionLine.Name
		case "vehicle_model":
			row[key] = p.VehicleModel.Name
		case "status":
			row[key] = statusLabel(p.Status)
		case "version":
			row[key] = p.Version
		case "description":
			row[key] = p.Description
		case "created_at":
			if !p.CreatedAt.IsZero() {
				row[key] = p.CreatedAt.Format(time.DateTime)
			} else {
				row[key] = ""
			}
		case "file_count":
			row[key] = p.OwnFileCount
		case "version_count":
			row[key] = p.OwnVersionCount
		default:
			if strings.HasPrefix(key, "cf_") {
				idStr := strings.TrimPrefix(key, "cf_")
				if id, err := strconv.ParseUint(idStr, 10, 64); err == nil {
					row[key] = cfMap[uint(id)]
				}
			}
		}
	}
	return row
}

// ──────────────────────────────────────────────────────────────
// 动态列 Excel 导出
// ──────────────────────────────────────────────────────────────

// ExportProgramsExcelDynamic 支持动态列选择的 Excel 导出。
func ExportProgramsExcelDynamic(c *gin.Context) {
	extraLineIDs := parseLineIDs(c)
	query, _, statusCode, msg := exportQueryPrograms(c, extraLineIDs)
	if statusCode != 0 {
		c.JSON(statusCode, gin.H{"error": msg})
		return
	}

	columnKeys := parseColumnKeys(c)
	includeStats := strings.TrimSpace(c.Query("include_stats")) == "true"

	var programs []models.Program
	if err := query.
		Preload("ProductionLine").
		Preload("VehicleModel").
		Preload("CustomFieldValues").
		Preload("CustomFieldValues.ProductionLineCustomField").
		Order("id DESC").
		Find(&programs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	// 加载所有程序的版本数和文件数
	programIDs := make([]uint, 0, len(programs))
	for _, p := range programs {
		programIDs = append(programIDs, p.ID)
	}
	versionCounts, _ := buildProgramVersionCountMap(database.DB, programIDs)
	fileCounts, _ := buildProgramFileCountMap(database.DB, programIDs)
	for i := range programs {
		programs[i].OwnVersionCount = versionCounts[programs[i].ID]
		programs[i].OwnFileCount = fileCounts[programs[i].ID]
	}

	// 查询自定义字段定义用于表头
	cfDefMap := loadCustomFieldDefMap()

	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheetName := "Programs"
	defaultSheet := f.GetSheetName(f.GetActiveSheetIndex())
	if defaultSheet != sheetName {
		_ = f.SetSheetName(defaultSheet, sheetName)
	}

	// 写表头
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#EBEEF0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	for i, key := range columnKeys {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		label := columnKeyToLabel(key, cfDefMap)
		_ = f.SetCellValue(sheetName, cell, label)
		_ = f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	// 写数据行
	for rowIdx, program := range programs {
		row := rowIdx + 2
		rowData := buildExportRow(program, columnKeys)
		for colIdx, key := range columnKeys {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, row)
			val := rowData[key]
			if val == nil {
				val = ""
			}
			_ = f.SetCellValue(sheetName, cell, val)
		}
	}

	// 自动列宽
	for i, key := range columnKeys {
		colName, _ := excelize.CoordinatesToCellName(i+1, 1)
		width := len(columnKeyToLabel(key, cfDefMap))*2 + 4
		if width < 12 {
			width = 12
		}
		if width > 40 {
			width = 40
		}
		_ = f.SetColWidth(sheetName, colName, colName, float64(width))
	}

	// 完成率统计 sheet
	if includeStats {
		writeCompletionStatsSheet(f, programs)
	}

	buffer, err := f.WriteToBuffer()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成Excel失败"})
		return
	}

	utf8FileName := "programs_export.xlsx"
	asciiFallback := "programs_export.xlsx"
	contentDisposition := "attachment; filename=\"" + asciiFallback + "\"; filename*=UTF-8''" + url.QueryEscape(utf8FileName)

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", contentDisposition)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer.Bytes())
}

func columnKeyToLabel(key string, cfDefMap map[uint]models.ProductionLineCustomField) string {
	for _, col := range builtinExportColumns() {
		if col.Key == key {
			return col.Label
		}
	}
	if strings.HasPrefix(key, "cf_") {
		idStr := strings.TrimPrefix(key, "cf_")
		if id, err := strconv.ParseUint(idStr, 10, 64); err == nil {
			if cf, ok := cfDefMap[uint(id)]; ok {
				return cf.Name
			}
		}
		return key
	}
	return key
}

// writeCompletionStatsSheet 生成产线×车型完成率统计 sheet。
func writeCompletionStatsSheet(f *excelize.File, programs []models.Program) {
	sheetName := "完成率统计"
	_, _ = f.NewSheet(sheetName)

	// 收集所有产线和车型
	lineSet := make(map[uint]string)
	modelSet := make(map[uint]string)
	cellMap := make(map[string]bool) // "lineID:modelID" -> has program

	for _, p := range programs {
		if p.ProductionLineID > 0 {
			lineSet[p.ProductionLineID] = p.ProductionLine.Name
		}
		if p.VehicleModelID > 0 {
			modelSet[p.VehicleModelID] = p.VehicleModel.Name
			key := fmt.Sprintf("%d:%d", p.ProductionLineID, p.VehicleModelID)
			cellMap[key] = true
		}
	}

	// 排序
	type idName struct {
		ID   uint
		Name string
	}
	lines := make([]idName, 0, len(lineSet))
	for id, name := range lineSet {
		lines = append(lines, idName{id, name})
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i].ID < lines[j].ID })

	vehicleModels := make([]idName, 0, len(modelSet))
	for id, name := range modelSet {
		vehicleModels = append(vehicleModels, idName{id, name})
	}
	sort.Slice(vehicleModels, func(i, j int) bool { return vehicleModels[i].ID < vehicleModels[j].ID })

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#EBEEF0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	// 表头：第一列"产线"，后续列各车型
	_ = f.SetCellValue(sheetName, "A1", "产线")
	_ = f.SetCellStyle(sheetName, "A1", "A1", headerStyle)
	for i, m := range vehicleModels {
		cell, _ := excelize.CoordinatesToCellName(i+2, 1)
		_ = f.SetCellValue(sheetName, cell, m.Name)
		_ = f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}
	// 汇总列
	totalCell, _ := excelize.CoordinatesToCellName(len(vehicleModels)+2, 1)
	_ = f.SetCellValue(sheetName, totalCell, "完成率")
	_ = f.SetCellStyle(sheetName, totalCell, totalCell, headerStyle)

	// 数据行
	for rowIdx, line := range lines {
		row := rowIdx + 2
		_ = f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), line.Name)

		completed := 0
		for colIdx, m := range vehicleModels {
			cell, _ := excelize.CoordinatesToCellName(colIdx+2, row)
			key := fmt.Sprintf("%d:%d", line.ID, m.ID)
			if cellMap[key] {
				_ = f.SetCellValue(sheetName, cell, "100%")
				completed++
			} else {
				_ = f.SetCellValue(sheetName, cell, "0%")
			}
		}

		// 完成率
		totalCell, _ := excelize.CoordinatesToCellName(len(vehicleModels)+2, row)
		if len(vehicleModels) > 0 {
			rate := float64(completed) / float64(len(vehicleModels)) * 100
			_ = f.SetCellValue(sheetName, totalCell, fmt.Sprintf("%.0f%%", rate))
		} else {
			_ = f.SetCellValue(sheetName, totalCell, "N/A")
		}
	}

	// 汇总行
	summaryRow := len(lines) + 2
	_ = f.SetCellValue(sheetName, fmt.Sprintf("A%d", summaryRow), "完成率")
	for colIdx, m := range vehicleModels {
		cell, _ := excelize.CoordinatesToCellName(colIdx+2, summaryRow)
		completed := 0
		for _, line := range lines {
			key := fmt.Sprintf("%d:%d", line.ID, m.ID)
			if cellMap[key] {
				completed++
			}
		}
		if len(lines) > 0 {
			rate := float64(completed) / float64(len(lines)) * 100
			_ = f.SetCellValue(sheetName, cell, fmt.Sprintf("%.0f%%", rate))
		}
	}

	// 总体完成率
	totalRow := summaryRow + 1
	_ = f.SetCellValue(sheetName, fmt.Sprintf("A%d", totalRow), "总体完成率")
	totalPairs := len(lines) * len(vehicleModels)
	totalCompleted := 0
	for _, line := range lines {
		for _, m := range vehicleModels {
			key := fmt.Sprintf("%d:%d", line.ID, m.ID)
			if cellMap[key] {
				totalCompleted++
			}
		}
	}
	if totalPairs > 0 {
		rate := float64(totalCompleted) / float64(totalPairs) * 100
		totalCell, _ := excelize.CoordinatesToCellName(len(vehicleModels)+2, totalRow)
		_ = f.SetCellValue(sheetName, totalCell, fmt.Sprintf("%.0f%%", rate))
	}

	// 列宽
	_ = f.SetColWidth(sheetName, "A", "A", 20)
	for i := range vehicleModels {
		colName, _ := excelize.CoordinatesToCellName(i+2, 1)
		_ = f.SetColWidth(sheetName, colName, colName, 15)
	}
}
