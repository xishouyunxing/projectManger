package controllers

import (
	"crane-system/migration"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetMigrationStatus 获取文件迁移状态
func GetMigrationStatus(c *gin.Context) {
	status := migration.GetMigrationStatus()
	c.JSON(http.StatusOK, status)
}

// StartMigration 开始文件迁移
func StartMigration(c *gin.Context) {
	// 检查是否已在运行
	status := migration.GetMigrationStatus()
	if status.Status == "running" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件迁移正在进行中"})
		return
	}

	// 异步执行迁移
	go func() {
		if err := migration.MigrateFilesToNewStructure(); err != nil {
			// 错误已经在迁移函数中处理，这里只是记录
			return
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "文件迁移已开始，请查看迁移状态",
		"status":  status,
	})
}

// RollbackMigration 回滚文件迁移
func RollbackMigration(c *gin.Context) {
	status := migration.GetMigrationStatus()
	if status.Status != "completed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只能回滚已完成的迁移"})
		return
	}

	// 异步执行回滚
	go func() {
		if err := migration.RollbackMigration(); err != nil {
			// 错误已经在回滚函数中处理
			return
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "迁移回滚已开始",
	})
}