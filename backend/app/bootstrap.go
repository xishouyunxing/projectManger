package app

import (
	"crane-system/config"
	"crane-system/database"
	"crane-system/router"
	"fmt"

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

func BootstrapServer() (*config.Config, *gin.Engine, error) {
	cfg, err := SetupInfrastructure()
	if err != nil {
		return nil, nil, err
	}

	if err := RunMigrationsIfEnabled(cfg); err != nil {
		return nil, nil, fmt.Errorf("run migrations: %w", err)
	}

	return cfg, BuildHTTPServer(), nil
}
