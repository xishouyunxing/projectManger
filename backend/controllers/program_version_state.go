package controllers

import (
	"crane-system/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func reconcileProgramVersionState(tx *gorm.DB, programID uint) error {
	var program models.Program
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&program, programID).Error; err != nil {
		return err
	}

	var files []models.ProgramFile
	if err := tx.Where("program_id = ?", programID).Order("created_at DESC, id DESC").Find(&files).Error; err != nil {
		return err
	}

	latestFileByVersion := make(map[string]models.ProgramFile, len(files))
	orderedVersions := make([]string, 0, len(files))
	for _, file := range files {
		if _, exists := latestFileByVersion[file.Version]; exists {
			continue
		}
		latestFileByVersion[file.Version] = file
		orderedVersions = append(orderedVersions, file.Version)
	}

	var versions []models.ProgramVersion
	if err := tx.Where("program_id = ?", programID).Order("is_current DESC, updated_at DESC, created_at DESC, id DESC").Find(&versions).Error; err != nil {
		return err
	}

	preferredCurrentVersion := ""
	knownVersionNames := make(map[string]struct{}, len(versions))
	for _, version := range versions {
		latestFile, hasFiles := latestFileByVersion[version.Version]
		if !hasFiles {
			if err := tx.Delete(&version).Error; err != nil {
				return err
			}
			continue
		}

		knownVersionNames[version.Version] = struct{}{}
		if version.IsCurrent && preferredCurrentVersion == "" {
			preferredCurrentVersion = version.Version
		}
		if version.FileID != latestFile.ID {
			if err := tx.Model(&models.ProgramVersion{}).Where("id = ?", version.ID).Update("file_id", latestFile.ID).Error; err != nil {
				return err
			}
		}
	}

	for _, versionName := range orderedVersions {
		if _, exists := knownVersionNames[versionName]; exists {
			continue
		}
		latestFile := latestFileByVersion[versionName]
		versionRecord := models.ProgramVersion{
			ProgramID:  programID,
			Version:    versionName,
			FileID:     latestFile.ID,
			UploadedBy: latestFile.UploadedBy,
			ChangeLog:  latestFile.Description,
			IsCurrent:  false,
		}
		if err := tx.Create(&versionRecord).Error; err != nil {
			return err
		}
	}

	if preferredCurrentVersion == "" {
		if program.Version != "" {
			if _, exists := latestFileByVersion[program.Version]; exists {
				preferredCurrentVersion = program.Version
			}
		}
		if preferredCurrentVersion == "" && len(orderedVersions) > 0 {
			preferredCurrentVersion = orderedVersions[0]
		}
	}

	if err := tx.Model(&models.ProgramVersion{}).Where("program_id = ?", programID).Update("is_current", false).Error; err != nil {
		return err
	}

	if preferredCurrentVersion != "" {
		var currentVersion models.ProgramVersion
		if err := tx.Where("program_id = ? AND version = ?", programID, preferredCurrentVersion).Order("created_at DESC, id DESC").First(&currentVersion).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.ProgramVersion{}).Where("id = ?", currentVersion.ID).Update("is_current", true).Error; err != nil {
			return err
		}
	}

	return tx.Model(&models.Program{}).Where("id = ?", programID).Update("version", preferredCurrentVersion).Error
}

func resolveDownloadVersion(tx *gorm.DB, programID uint) (models.ProgramVersion, error) {
	var version models.ProgramVersion
	if err := tx.Where("program_id = ? AND is_current = ?", programID, true).Order("updated_at DESC, created_at DESC, id DESC").First(&version).Error; err == nil {
		return version, nil
	} else if err != gorm.ErrRecordNotFound {
		return models.ProgramVersion{}, err
	}

	if err := tx.Where("program_id = ?", programID).Order("created_at DESC, id DESC").First(&version).Error; err != nil {
		return models.ProgramVersion{}, err
	}
	return version, nil
}
