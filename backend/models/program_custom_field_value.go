package models

import (
	"time"
)

type ProgramCustomFieldValue struct {
	ID                          uint      `gorm:"primarykey" json:"id"`
	CreatedAt                   time.Time `json:"created_at"`
	UpdatedAt                   time.Time `json:"updated_at"`
	ProgramID                   uint      `gorm:"not null;uniqueIndex:idx_program_custom_field_values_program_field" json:"program_id"`
	ProductionLineCustomFieldID uint      `gorm:"not null;uniqueIndex:idx_program_custom_field_values_program_field" json:"production_line_custom_field_id"`
	Value                       string    `gorm:"type:text" json:"value"`

	Program                   Program                   `json:"program,omitempty"`
	ProductionLineCustomField ProductionLineCustomField `json:"production_line_custom_field,omitempty"`
}
