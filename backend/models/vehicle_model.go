package models

import (
	"time"

	"gorm.io/gorm"
)

type VehicleModel struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:200;not null" json:"name"`            // 车型名称
	Code        string         `gorm:"uniqueIndex;size:50;not null" json:"code"` // 车型编号
	Series      string         `gorm:"size:100" json:"series"`                   // 系列
	Description string         `gorm:"type:text" json:"description"`             // 描述
	Status      string         `gorm:"size:20;default:active" json:"status"`     // 状态

	// 关联
	Programs []Program `json:"programs,omitempty"`
}
