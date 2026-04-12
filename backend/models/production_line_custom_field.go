package models

import (
	"time"
)

type ProductionLineCustomField struct {
	ID               uint      `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	ProductionLineID uint      `gorm:"not null;uniqueIndex:idx_production_line_custom_fields_line_name" json:"production_line_id"`
	Name             string    `gorm:"size:100;not null;uniqueIndex:idx_production_line_custom_fields_line_name" json:"name"`
	FieldType        string    `gorm:"size:20;not null" json:"field_type"`
	OptionsJSON      string    `gorm:"type:text" json:"options_json"`
	SortOrder        int       `gorm:"default:0" json:"sort_order"`
	Enabled          bool      `gorm:"default:true" json:"enabled"`

	ProductionLine ProductionLine            `json:"production_line,omitempty"`
	Values         []ProgramCustomFieldValue `json:"values,omitempty"`
}
