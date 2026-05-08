package models

import (
	"time"

	"gorm.io/gorm"
)

// Role 是权限系统的核心实体。
// 预设角色（is_preset=true）不可删除，系统角色（is_system=true）不可删除不可改名。
// 每个用户通过 role_id 关联一个角色，角色层级决定基础权限。
type Role struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:50;not null;uniqueIndex" json:"name"`
	Description string         `gorm:"size:200" json:"description"`
	IsPreset    bool           `gorm:"not null;default:false" json:"is_preset"`
	IsSystem    bool           `gorm:"not null;default:false" json:"is_system"`
	Status      string         `gorm:"size:20;not null;default:active" json:"status"`
	SortOrder   int            `gorm:"not null;default:0" json:"sort_order"`

	Permissions    []Permission         `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
	LinePermissions []RoleLinePermission `json:"line_permissions,omitempty"`
}
