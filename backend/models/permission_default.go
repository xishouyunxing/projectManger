package models

import (
	"time"

	"gorm.io/gorm"
)

// RoleDefaultPermission 是角色级默认权限。
// 它只在用户和部门都没有显式配置时参与权限解析，适合定义普通角色的基础可见范围。
type RoleDefaultPermission struct {
	ID               uint           `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	Role             string         `gorm:"size:50;not null;uniqueIndex:idx_role_default_line" json:"role"`
	ProductionLineID uint           `gorm:"not null;uniqueIndex:idx_role_default_line" json:"production_line_id"`
	CanView          bool           `gorm:"default:false" json:"can_view"`
	CanDownload      bool           `gorm:"default:false" json:"can_download"`
	CanUpload        bool           `gorm:"default:false" json:"can_upload"`
	CanManage        bool           `gorm:"default:false" json:"can_manage"`

	ProductionLine ProductionLine `json:"production_line,omitempty"`
}

// DepartmentDefaultPermission 是部门级默认权限。
// 它用于给部门成员提供兜底权限，不应覆盖用户或部门的显式授权/拒绝。
type DepartmentDefaultPermission struct {
	ID               uint           `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	DepartmentID     uint           `gorm:"not null;uniqueIndex:idx_department_default_line" json:"department_id"`
	ProductionLineID uint           `gorm:"not null;uniqueIndex:idx_department_default_line" json:"production_line_id"`
	CanView          bool           `gorm:"default:false" json:"can_view"`
	CanDownload      bool           `gorm:"default:false" json:"can_download"`
	CanUpload        bool           `gorm:"default:false" json:"can_upload"`
	CanManage        bool           `gorm:"default:false" json:"can_manage"`

	Department     Department     `json:"department,omitempty"`
	ProductionLine ProductionLine `json:"production_line,omitempty"`
}
