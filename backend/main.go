package main

import (
	"crane-system/config"
	"crane-system/database"
	"crane-system/router"
	"log"
)

func main() {
	// 加载配置
	config.LoadConfig()

	// 连接数据库
	if err := database.Connect(); err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	// 自动迁移
	if err := database.AutoMigrate(); err != nil {
		log.Fatal("数据库迁移失败:", err)
	}

	// 初始化路由
	r := router.SetupRouter()

	// 启动服务器
	port := config.AppConfig.ServerPort
	log.Printf("服务器启动在端口 %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}
