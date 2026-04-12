package models

import (
	"time"

	"gorm.io/gorm"
)

type Department struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	Name        string         `gorm:"size:100;not null;unique" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Status      string         `gorm:"size:20;default:active" json:"status"`
}
