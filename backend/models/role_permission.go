package models

import "time"

// RolePermission 是角色与功能权限的关联表。
// 一个角色可以有多个功能权限，一个功能权限可以属于多个角色。
type RolePermission struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	RoleID       uint      `gorm:"not null;uniqueIndex:idx_role_permission" json:"role_id"`
	PermissionID uint      `gorm:"not null;uniqueIndex:idx_role_permission" json:"permission_id"`
	CreatedAt    time.Time `json:"created_at"`
}
