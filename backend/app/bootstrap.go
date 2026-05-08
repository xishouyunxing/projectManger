package app

import (
	"crane-system/config"
	"crane-system/database"
	"crane-system/router"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func LoadConfig() (*config.Config, error) {
	if err := config.LoadConfig(); err != nil {
		return nil, err
	}

	return config.AppConfig, nil
}

func SetupInfrastructure() (*config.Config, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	if err := EnsureRuntimeDirs(cfg); err != nil {
		return nil, err
	}

	if err := database.Connect(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func RunMigrationsIfEnabled(cfg *config.Config) error {
	if !cfg.App.AutoMigrate {
		return nil
	}

	return database.AutoMigrate()
}

func BuildHTTPServer() *gin.Engine {
	return router.SetupRouter()
}

func ServerAddress(cfg *config.Config) string {
	return ":" + cfg.App.ServerPort
}

// BuildAppServer 创建带超时配置的 http.Server，支持优雅关闭。
func BuildAppServer(engine *gin.Engine, addr string) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB
	}
}

func BootstrapServer() (*config.Config, *http.Server, error) {
	cfg, err := SetupInfrastructure()
	if err != nil {
		return nil, nil, err
	}

	if err := RunMigrationsIfEnabled(cfg); err != nil {
		return nil, nil, fmt.Errorf("run migrations: %w", err)
	}

	engine := BuildHTTPServer()
	srv := BuildAppServer(engine, ServerAddress(cfg))
	return cfg, srv, nil
}
