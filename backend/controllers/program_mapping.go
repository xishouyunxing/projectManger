package controllers

import (
	"crane-system/database"
	"crane-system/models"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type createProgramMappingsRequest struct {
	ChildProgramIDs []uint `json:"child_program_ids"`
}

type programMappingItem struct {
	ID        uint           `json:"id"`
	Parent    models.Program `json:"parent_program"`
	Child     models.Program `json:"child_program"`
	CreatedBy uint           `json:"created_by"`
	CreatedAt string         `json:"created_at"`
}

func resolveProgramTarget(tx *gorm.DB, programID uint) (models.Program, uint, *models.ProgramMapping, error) {
	var program models.Program
	if err := tx.Preload("ProductionLine").Preload("VehicleModel").First(&program, programID).Error; err != nil {
		return models.Program{}, 0, nil, err
	}

	var mapping models.ProgramMapping
	if err := tx.Preload("ParentProgram").Where("child_program_id = ?", programID).First(&mapping).Error; err == nil {
		var parent models.Program
		if err := tx.Preload("ProductionLine").Preload("VehicleModel").First(&parent, mapping.ParentProgramID).Error; err != nil {
			return models.Program{}, 0, nil, err
		}
		program.MappingInfo = &models.ProgramMappingInfo{
			MappingID:         mapping.ID,
			ParentProgramID:   parent.ID,
			ParentProgramName: parent.Name,
			ParentProgramCode: parent.Code,
		}
		return parent, parent.ID, &mapping, nil
	}
	if err := tx.Where("parent_program_id = ?", programID).First(&mapping).Error; err == nil {
		return program, program.ID, nil, nil
	}
	if err := tx.Where("child_program_id = ?", programID).First(&mapping).Error; err == nil {
		return program, program.ID, &mapping, nil
	}
	return program, program.ID, nil, nil
}

func attachProgramMappingInfo(tx *gorm.DB, program *models.Program) error {
	var versionCount int64
	if err := tx.Model(&models.ProgramVersion{}).Where("program_id = ?", program.ID).Count(&versionCount).Error; err != nil {
		return err
	}
	program.OwnVersionCount = versionCount

	var fileCount int64
	if err := tx.Model(&models.ProgramFile{}).Where("program_id = ?", program.ID).Count(&fileCount).Error; err != nil {
		return err
	}
	program.OwnFileCount = fileCount

	var mapping models.ProgramMapping
	if err := tx.Preload("ParentProgram").Where("child_program_id = ?", program.ID).First(&mapping).Error; err == nil {
		program.MappingInfo = &models.ProgramMappingInfo{
			MappingID:         mapping.ID,
			ParentProgramID:   mapping.ParentProgram.ID,
			ParentProgramName: mapping.ParentProgram.Name,
			ParentProgramCode: mapping.ParentProgram.Code,
		}
	}

	return nil
}

func applyParentProgramData(child *models.Program, parent models.Program) {
	child.ProductionLineID = parent.ProductionLineID
	child.VehicleModelID = parent.VehicleModelID
	child.Version = parent.Version
	child.Description = parent.Description
	child.Status = parent.Status
	child.ProductionLine = parent.ProductionLine
	child.VehicleModel = parent.VehicleModel
	child.Files = parent.Files
	child.Versions = parent.Versions
	child.CustomFieldValues = parent.CustomFieldValues
}

func GetProgramMappingsByParent(c *gin.Context) {
	parentProgramID, err := parseUintParam(c.Param("program_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??ID????"})
		return
	}
	if !authorizeProgramAction(c, database.DB, parentProgramID, lineActionView) {
		return
	}

	var mappings []models.ProgramMapping
	if err := database.DB.
		Preload("ParentProgram").
		Preload("ChildProgram").
		Where("parent_program_id = ?", parentProgramID).
		Order("created_at DESC").
		Find(&mappings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "??????"})
		return
	}

	filteredMappings := make([]models.ProgramMapping, 0, len(mappings))
	for _, mapping := range mappings {
		allowed, statusCode, message := checkLineAction(c, mapping.ChildProgram.ProductionLineID, lineActionView)
		if !allowed {
			if statusCode == http.StatusForbidden {
				continue
			}
			c.JSON(statusCode, gin.H{"error": message})
			return
		}
		filteredMappings = append(filteredMappings, mapping)
	}
	mappings = filteredMappings

	childProgramIDs := make([]uint, 0, len(mappings))
	for _, mapping := range mappings {
		childProgramIDs = append(childProgramIDs, mapping.ChildProgram.ID)
	}
	versionCounts, err := buildProgramVersionCountMap(database.DB, childProgramIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "??????"})
		return
	}
	fileCounts, err := buildProgramFileCountMap(database.DB, childProgramIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "??????"})
		return
	}

	for i := range mappings {
		child := &mappings[i].ChildProgram
		child.OwnVersionCount = versionCounts[child.ID]
		child.OwnFileCount = fileCounts[child.ID]
		child.MappingInfo = &models.ProgramMappingInfo{
			MappingID:         mappings[i].ID,
			ParentProgramID:   mappings[i].ParentProgram.ID,
			ParentProgramName: mappings[i].ParentProgram.Name,
			ParentProgramCode: mappings[i].ParentProgram.Code,
		}
	}

	c.JSON(http.StatusOK, mappings)
}

func GetProgramMappingByChild(c *gin.Context) {
	childProgramID, err := parseUintParam(c.Param("program_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??ID????"})
		return
	}
	if !authorizeProgramAction(c, database.DB, childProgramID, lineActionView) {
		return
	}

	var mapping models.ProgramMapping
	if err := database.DB.
		Preload("ParentProgram").
		Preload("ChildProgram").
		Where("child_program_id = ?", childProgramID).
		First(&mapping).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "??????"})
		return
	}

	if err := attachProgramMappingInfo(database.DB, &mapping.ChildProgram); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "??????"})
		return
	}

	allowed, statusCode, message := checkLineAction(c, mapping.ParentProgram.ProductionLineID, lineActionView)
	if !allowed {
		if statusCode == http.StatusForbidden {
			c.JSON(http.StatusOK, gin.H{})
			return
		}
		c.JSON(statusCode, gin.H{"error": message})
		return
	}

	c.JSON(http.StatusOK, mapping)
}

func CreateProgramMappings(c *gin.Context) {
	var req createProgramMappingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.ChildProgramIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "child_program_ids ????"})
		return
	}

	parentProgramIDValue, ok := c.GetQuery("parent_program_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "?? parent_program_id ??"})
		return
	}
	parentProgramID, err := parseUintParam(parentProgramIDValue)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "???ID????"})
		return
	}

	if !authorizeProgramAction(c, database.DB, parentProgramID, lineActionManage) {
		return
	}

	var parentProgram models.Program
	if err := database.DB.First(&parentProgram, parentProgramID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "??????"})
		return
	}

	userID, _ := c.Get("user_id")

	createdMappings := make([]models.ProgramMapping, 0, len(req.ChildProgramIDs))
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, childID := range req.ChildProgramIDs {
			if childID == 0 || childID == parentProgram.ID {
				return errors.New("??????????")
			}

			var childProgram models.Program
			if err := tx.First(&childProgram, childID).Error; err != nil {
				return errors.New("??????")
			}
			allowed, _, _ := checkLineAction(c, childProgram.ProductionLineID, lineActionManage)
			if !allowed {
				return errors.New("forbidden_child_program")
			}

			var existingMapping models.ProgramMapping
			if err := tx.Where("child_program_id = ?", childID).First(&existingMapping).Error; err == nil {
				return errors.New("????????")
			}

			var versionCount int64
			if err := tx.Model(&models.ProgramVersion{}).Where("program_id = ?", childID).Count(&versionCount).Error; err != nil {
				return err
			}
			var fileCount int64
			if err := tx.Model(&models.ProgramFile{}).Where("program_id = ?", childID).Count(&fileCount).Error; err != nil {
				return err
			}
			if versionCount > 0 || fileCount > 0 {
				return errors.New("??????????????")
			}

			mapping := models.ProgramMapping{
				ParentProgramID: parentProgram.ID,
				ChildProgramID:  childID,
				CreatedBy:       userID.(uint),
			}
			if err := tx.Create(&mapping).Error; err != nil {
				return err
			}
			createdMappings = append(createdMappings, mapping)
		}
		return nil
	}); err != nil {
		if err.Error() == "forbidden_child_program" {
			c.JSON(http.StatusForbidden, gin.H{"error": "?????????????"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"mappings": createdMappings})
}

func DeleteProgramMapping(c *gin.Context) {
	mappingID, err := parseUintParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "??ID????"})
		return
	}

	var mapping models.ProgramMapping
	if err := database.DB.First(&mapping, mappingID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "?????"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "??????"})
		return
	}
	if !authorizeProgramAction(c, database.DB, mapping.ParentProgramID, lineActionManage) {
		return
	}
	if !authorizeProgramAction(c, database.DB, mapping.ChildProgramID, lineActionManage) {
		return
	}

	result := database.DB.Delete(&models.ProgramMapping{}, mappingID)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "??????"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "??????"})
}
