package main

import (
	"context"
	"crane-system/app"
	"crane-system/database"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 初始化结构化日志
	initLogger()

	cfg, srv, err := app.BootstrapServer()
	if err != nil {
		slog.Error("服务启动准备失败", "error", err)
		log.Fatal("服务启动准备失败:", err)
	}

	// 在 goroutine 中启动服务
	go func() {
		slog.Info("服务器启动", "port", cfg.App.ServerPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("服务器异常退出", "error", err)
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 监听 SIGINT / SIGTERM 信号，优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	slog.Info("收到关闭信号，正在优雅关闭", "signal", sig.String())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("HTTP 服务关闭失败", "error", err)
	} else {
		slog.Info("HTTP 服务已关闭")
	}

	if err := database.Close(); err != nil {
		slog.Error("数据库连接关闭失败", "error", err)
	} else {
		slog.Info("数据库连接已关闭")
	}

	slog.Info("服务已完全停止")
}

// initLogger 根据 APP_ENV 配置 slog。
// production: JSON 格式，Warn 级别
// development: 文本格式，Debug 级别
func initLogger() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}

	slog.SetDefault(slog.New(handler))
}
