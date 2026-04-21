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
	ID           uint           `json:"id"`
	Parent       models.Program `json:"parent_program"`
	Child        models.Program `json:"child_program"`
	CreatedBy    uint           `json:"created_by"`
	CreatedAt    string         `json:"created_at"`
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
			MappingID:          mapping.ID,
			ParentProgramID:    mapping.ParentProgram.ID,
			ParentProgramName:  mapping.ParentProgram.Name,
			ParentProgramCode:  mapping.ParentProgram.Code,
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
	var mappings []models.ProgramMapping
	if err := database.DB.
		Preload("ParentProgram").
		Preload("ChildProgram").
		Where("parent_program_id = ?", c.Param("program_id")).
		Order("created_at DESC").
		Find(&mappings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询映射失败"})
		return
	}

	for i := range mappings {
		if err := attachProgramMappingInfo(database.DB, &mappings[i].ChildProgram); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询映射失败"})
			return
		}
	}

	c.JSON(http.StatusOK, mappings)
}

func GetProgramMappingByChild(c *gin.Context) {
	var mapping models.ProgramMapping
	if err := database.DB.
		Preload("ParentProgram").
		Preload("ChildProgram").
		Where("child_program_id = ?", c.Param("program_id")).
		First(&mapping).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询映射失败"})
		return
	}

	if err := attachProgramMappingInfo(database.DB, &mapping.ChildProgram); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询映射失败"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "child_program_ids 不能为空"})
		return
	}

	parentProgramIDValue, ok := c.GetQuery("parent_program_id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 parent_program_id 参数"})
		return
	}

	var parentProgram models.Program
	if err := database.DB.First(&parentProgram, parentProgramIDValue).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "父程序不存在"})
		return
	}

	userID, _ := c.Get("user_id")

	createdMappings := make([]models.ProgramMapping, 0, len(req.ChildProgramIDs))
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, childID := range req.ChildProgramIDs {
			if childID == 0 || childID == parentProgram.ID {
				return errors.New("不能将程序映射到自己")
			}

			var childProgram models.Program
			if err := tx.First(&childProgram, childID).Error; err != nil {
				return errors.New("子程序不存在")
			}

			var existingMapping models.ProgramMapping
			if err := tx.Where("child_program_id = ?", childID).First(&existingMapping).Error; err == nil {
				return errors.New("所选程序已被映射")
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
				return errors.New("被映射程序下必须没有任何版本")
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"mappings": createdMappings})
}

func DeleteProgramMapping(c *gin.Context) {
	if err := database.DB.Delete(&models.ProgramMapping{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "取消映射失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "取消映射成功"})
}
