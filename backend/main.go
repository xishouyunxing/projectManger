package main

import (
	"crane-system/app"
	"log"
)

func main() {
	cfg, server, err := app.BootstrapServer()
	if err != nil {
		log.Fatal("服务启动准备失败:", err)
	}

	addr := app.ServerAddress(cfg)
	log.Printf("服务器启动在端口 %s", cfg.App.ServerPort)
	if err := server.Run(addr); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}
