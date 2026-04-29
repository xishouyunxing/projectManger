package models

import "time"

// UserPermissionOverride 是用户对功能权限的覆盖记录。
// granted=true 表示授予（即使角色没有该权限），granted=false 表示显式拒绝。
// 优先级高于角色功能权限。
type UserPermissionOverride struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	UserID       uint      `gorm:"not null;uniqueIndex:idx_user_permission" json:"user_id"`
	PermissionID uint      `gorm:"not null;uniqueIndex:idx_user_permission" json:"permission_id"`
	Granted      bool      `gorm:"not null" json:"granted"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	Permission Permission `json:"permission,omitempty"`
}
