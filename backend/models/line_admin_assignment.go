package models

import "time"

// LineAdminAssignment 记录产线管理员与产线的绑定关系。
// 一个用户可以管理多条产线，一条产线可以有多个管理员（交叉管理）。
type LineAdminAssignment struct {
	ID               uint      `gorm:"primarykey" json:"id"`
	UserID           uint      `gorm:"not null;uniqueIndex:idx_line_admin" json:"user_id"`
	ProductionLineID uint      `gorm:"not null;uniqueIndex:idx_line_admin" json:"production_line_id"`
	CreatedBy        *uint     `json:"created_by"`
	CreatedAt        time.Time `json:"created_at"`

	User           User           `json:"user,omitempty"`
	ProductionLine ProductionLine `json:"production_line,omitempty"`
}
