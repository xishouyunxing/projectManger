package models

import (
	"time"

	"gorm.io/gorm"
)

// ProductionLine 是权限控制和程序归属的核心业务边界。
// 多数程序、文件和矩阵权限最终都会落到生产线维度做授权判断。
type ProductionLine struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:200;not null" json:"name"`            // 生产线名称
	Code        string         `gorm:"uniqueIndex;size:50;not null" json:"code"` // 生产线编号
	Type        string         `gorm:"size:50;not null" json:"type"`             // 类型: upper(上车), lower(下车)
	Description string         `gorm:"type:text" json:"description"`             // 描述
	Status      string         `gorm:"size:20;default:active" json:"status"`     // 状态: active, inactive
	ProcessID   *uint          `gorm:"index" json:"process_id"`                  // 所属工序，可为空

	// 关联
	Process              *Process                    `json:"process,omitempty"`
	Programs             []Program                   `json:"programs,omitempty"`
	CustomFieldTemplates []ProductionLineCustomField `json:"custom_field_templates,omitempty"`
}

// Process 表示工序分组，用于组织生产线。
// 删除工序前需要确保没有生产线仍依赖它。
type Process struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:200;not null" json:"name"`            // 工序名称
	Code        string         `gorm:"uniqueIndex;size:50;not null" json:"code"` // 工序编号
	Type        string         `gorm:"size:50;not null" json:"type"`             // 类型: upper(上车), lower(下车)
	SortOrder   int            `gorm:"default:0" json:"sort_order"`              // 排序
	Description string         `gorm:"type:text" json:"description"`             // 描述

	// 关联
	ProductionLines []ProductionLine `json:"production_lines,omitempty"`
}
