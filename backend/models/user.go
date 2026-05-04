package models

import (
	"time"

	"gorm.io/gorm"
)

// User 是系统登录账号，也是权限解析的主体。
// 通过 role_id 关联 Role 表获取基础权限，user_permissions 表存储产线权限覆盖。
type User struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	EmployeeID   string         `gorm:"unique;size:50" json:"employee_id"`
	EmployeeNo   string         `gorm:"size:50" json:"employee_no"`
	Name         string         `gorm:"size:100;not null" json:"name"`
	DepartmentID *uint          `gorm:"index" json:"department_id"`
	Role         string         `gorm:"size:50;not null" json:"role"`
	RoleID       *uint          `gorm:"index" json:"role_id"`
	Password     string         `gorm:"size:255;not null" json:"-"`
	Status       string         `gorm:"size:20;default:active" json:"status"`

	Department *Department `json:"department,omitempty"`
	RoleRef    *Role       `gorm:"foreignKey:RoleID" json:"role_ref,omitempty"`
}

// UserPermission 表示“某用户 + 某产线”的显式权限覆盖。
// 注意：四个权限位都允许为 false，用来表达管理员明确拒绝该用户访问该产线。
type UserPermission struct {
	ID               uint           `gorm:"primarykey" json:"id"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
	UserID           uint           `gorm:"not null;index;uniqueIndex:idx_user_line" json:"user_id"`
	ProductionLineID uint           `gorm:"not null;index;uniqueIndex:idx_user_line" json:"production_line_id"`
	CanView          bool           `gorm:"default:true" json:"can_view"`      // 查看权限
	CanDownload      bool           `gorm:"default:false" json:"can_download"` // 下载权限
	CanUpload        bool           `gorm:"default:false" json:"can_upload"`   // 上传权限
	CanManage        bool           `gorm:"default:false" json:"can_manage"`   // 管理权限

	// 关联
	User           User           `json:"user,omitempty"`
	ProductionLine ProductionLine `json:"production_line,omitempty"`
}
