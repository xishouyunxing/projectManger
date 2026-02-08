package models

import (
	"time"

	"gorm.io/gorm"
)

type Program struct {
	ID               uint           `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	Name             string         `gorm:"size:200;not null" json:"name"`            // 程序名称
	Code             string         `gorm:"size:100;not null" json:"code"`            // 程序编号
	ProductionLineID uint           `gorm:"not null;index" json:"production_line_id"` // 生产线ID
	VehicleModelID   uint           `gorm:"index" json:"vehicle_model_id"`            // 车型ID
	Version          string         `gorm:"size:50" json:"version"`                   // 当前版本
	Description      string         `gorm:"type:text" json:"description"`             // 描述
	Status           string         `gorm:"size:20;default:active" json:"status"`     // 状态

	// 关联
	ProductionLine ProductionLine    `json:"production_line,omitempty"`
	VehicleModel   VehicleModel      `json:"vehicle_model,omitempty"`
	Files          []ProgramFile     `json:"files,omitempty"`
	Versions       []ProgramVersion  `json:"versions,omitempty"`
}

type ProgramFile struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	ProgramID   uint           `gorm:"not null;index" json:"program_id"`    // 程序ID
	FileName    string         `gorm:"size:255;not null" json:"file_name"`  // 文件名
	FilePath    string         `gorm:"size:500;not null" json:"file_path"`  // 文件路径
	FileSize    int64          `json:"file_size"`                           // 文件大小(字节)
	FileType    string         `gorm:"size:50" json:"file_type"`            // 文件类型
	Version     string         `gorm:"size:50" json:"version"`              // 版本号
	UploadedBy  uint           `gorm:"index" json:"uploaded_by"`            // 上传人ID
	Description string         `gorm:"type:text" json:"description"`        // 描述

	// 关联
	Program  Program `json:"program,omitempty"`
	Uploader User    `gorm:"foreignKey:UploadedBy" json:"uploader,omitempty"`
}

type ProgramVersion struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	ProgramID   uint           `gorm:"not null;index" json:"program_id"`    // 程序ID
	Version     string         `gorm:"size:50;not null" json:"version"`     // 版本号
	FileID      uint           `gorm:"not null" json:"file_id"`             // 文件ID
	UploadedBy  uint           `gorm:"index" json:"uploaded_by"`            // 上传人ID
	ChangeLog   string         `gorm:"type:text" json:"change_log"`         // 变更日志
	IsCurrent   bool           `gorm:"default:false" json:"is_current"`     // 是否当前版本

	// 关联
	Program  Program     `json:"program,omitempty"`
	File     ProgramFile `gorm:"foreignKey:FileID" json:"file,omitempty"`
	Uploader User        `gorm:"foreignKey:UploadedBy" json:"uploader,omitempty"`
}

type ProgramRelation struct {
	ID               uint           `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	SourceProgramID  uint           `gorm:"not null;index" json:"source_program_id"`  // 源程序ID
	RelatedProgramID uint           `gorm:"not null;index" json:"related_program_id"` // 关联程序ID
	RelationType     string         `gorm:"size:50" json:"relation_type"`             // 关系类型: same_program(相同程序)
	Description      string         `gorm:"type:text" json:"description"`             // 关系描述

	// 关联
	SourceProgram  Program `gorm:"foreignKey:SourceProgramID" json:"source_program,omitempty"`
	RelatedProgram Program `gorm:"foreignKey:RelatedProgramID" json:"related_program,omitempty"`
}
