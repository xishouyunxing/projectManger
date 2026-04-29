package models

// Permission 表示一个功能权限项。
// type 为 page 时表示页面访问权限，为 operation 时表示操作权限。
// code 是唯一标识，格式为 "page:xxx" 或 "op:xxx"。
type Permission struct {
	ID          uint   `gorm:"primarykey" json:"id"`
	Code        string `gorm:"size:100;not null;uniqueIndex" json:"code"`
	Name        string `gorm:"size:100;not null" json:"name"`
	Type        string `gorm:"size:20;not null" json:"type"`
	Resource    string `gorm:"size:100" json:"resource"`
	Description string `gorm:"size:200" json:"description"`
}
