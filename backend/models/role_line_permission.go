package models

import "time"

// RoleLinePermission 是角色对产线的权限配置。
// 替代原有的 RoleDefaultPermission，绑定到 role_id 而非 role 字符串。
type RoleLinePermission struct {
	ID               uint      `gorm:"primarykey" json:"id"`
	RoleID           uint      `gorm:"not null;uniqueIndex:idx_role_line" json:"role_id"`
	ProductionLineID uint      `gorm:"not null;uniqueIndex:idx_role_line" json:"production_line_id"`
	CanView          bool      `gorm:"not null;default:false" json:"can_view"`
	CanDownload      bool      `gorm:"not null;default:false" json:"can_download"`
	CanUpload        bool      `gorm:"not null;default:false" json:"can_upload"`
	CanManage        bool      `gorm:"not null;default:false" json:"can_manage"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`

	ProductionLine ProductionLine `json:"production_line,omitempty"`
}
