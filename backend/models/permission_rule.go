package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	PermissionSubjectUser              = "user"
	PermissionSubjectDepartment        = "department"
	PermissionSubjectRole              = "role"
	PermissionSubjectDepartmentDefault = "department_default"

	PermissionResourceProductionLine = "production_line"
	PermissionResourceSystem         = "system"

	PermissionDecisionAllow = "allow"
	PermissionDecisionDeny  = "deny"

	PermissionActionView     = "view"
	PermissionActionDownload = "download"
	PermissionActionUpload   = "upload"
	PermissionActionManage   = "manage"
)

// PermissionRule stores one explicit allow/deny decision for one subject,
// resource, and action. Missing rows are shown as "按规则" in the UI.
type PermissionRule struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	SubjectType  string         `gorm:"size:40;not null;uniqueIndex:idx_permission_rule_scope" json:"subject_type"`
	SubjectID    uint           `gorm:"not null;uniqueIndex:idx_permission_rule_scope" json:"subject_id"`
	SubjectKey   string         `gorm:"size:100;not null;default:'';uniqueIndex:idx_permission_rule_scope" json:"subject_key"`
	ResourceType string         `gorm:"size:40;not null;uniqueIndex:idx_permission_rule_scope" json:"resource_type"`
	ResourceID   uint           `gorm:"not null;default:0;uniqueIndex:idx_permission_rule_scope" json:"resource_id"`
	Action       string         `gorm:"size:80;not null;uniqueIndex:idx_permission_rule_scope" json:"action"`
	Decision     string         `gorm:"size:20;not null" json:"decision"`
}
