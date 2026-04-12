package models

import (
	"time"

	"gorm.io/gorm"
)

type DepartmentPermission struct {
	ID               uint           `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	DepartmentID     uint           `gorm:"not null;uniqueIndex:idx_department_line" json:"department_id"`
	ProductionLineID uint           `gorm:"not null;uniqueIndex:idx_department_line" json:"production_line_id"`
	CanView          bool           `gorm:"default:true" json:"can_view"`
	CanDownload      bool           `gorm:"default:false" json:"can_download"`
	CanUpload        bool           `gorm:"default:false" json:"can_upload"`
	CanManage        bool           `gorm:"default:false" json:"can_manage"`

	Department     Department     `json:"department,omitempty"`
	ProductionLine ProductionLine `json:"production_line,omitempty"`
}
