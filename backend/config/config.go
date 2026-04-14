package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type AppSection struct {
	Env          string
	ServerPort   string
	FrontendDist string
	AutoMigrate  bool
}

type DatabaseSection struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type AuthSection struct {
	JWTSecret       string
	DefaultPassword string
}

type StorageSection struct {
	UploadsDir    string
	MaxUploadSize int64
}

type BackupSection struct {
	Dir string
}

type CORSSection struct {
	AllowedOrigins []string
}

type Config struct {
	App      AppSection
	Database DatabaseSection
	Auth     AuthSection
	Storage  StorageSection
	Backup   BackupSection
	CORS     CORSSection
}

var AppConfig *Config

func LoadConfig() error {
	envPaths := []string{
		"../.env",
		".env",
	}

	envLoaded := false
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			log.Printf("成功加载.env文件: %s", path)
			envLoaded = true
			break
		}
	}

	if !envLoaded {
		log.Println("未找到.env文件,使用环境变量配置")
	}

	appEnv := getEnv("APP_ENV", "development")
	corsAllowedOrigins := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000")

	AppConfig = &Config{
		App: AppSection{
			Env:          appEnv,
			ServerPort:   getEnv("SERVER_PORT", "8080"),
			FrontendDist: getEnv("FRONTEND_DIST", "../frontend/dist"),
			AutoMigrate:  getEnvBool("AUTO_MIGRATE", appEnv != "production"),
		},
		Database: DatabaseSection{
			Host:     getEnv("DB_HOST", "127.0.0.1"),
			Port:     getEnv("DB_PORT", "3307"),
			User:     getEnv("DB_USER", "crane_user"),
			Password: getEnv("DB_PASSWORD", "zlzk.12345678"),
			Name:     getEnv("DB_NAME", "crane_system"),
		},
		Auth: AuthSection{
			JWTSecret:       os.Getenv("JWT_SECRET"),
			DefaultPassword: os.Getenv("DEFAULT_PASSWORD"),
		},
		Storage: StorageSection{
			UploadsDir:    cleanPath(getEnv("UPLOADS_DIR", "./uploads")),
			MaxUploadSize: 100 * 1024 * 1024,
		},
		Backup: BackupSection{
			Dir: cleanPath(getEnv("BACKUPS_DIR", "./backups")),
		},
		CORS: CORSSection{
			AllowedOrigins: splitCSV(corsAllowedOrigins),
		},
	}

	return validateConfig(AppConfig)
}

func validateConfig(cfg *Config) error {
	if strings.TrimSpace(cfg.Auth.JWTSecret) == "" {
		return fmt.Errorf("JWT_SECRET 未配置")
	}

	if len(cfg.Auth.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET 长度不能少于 32 个字符")
	}

	if strings.TrimSpace(cfg.Auth.DefaultPassword) == "" {
		return fmt.Errorf("DEFAULT_PASSWORD 未配置")
	}

	if len(cfg.Auth.DefaultPassword) < 8 {
		return fmt.Errorf("DEFAULT_PASSWORD 长度不能少于 8 个字符")
	}

	if strings.TrimSpace(cfg.Storage.UploadsDir) == "" {
		return fmt.Errorf("UPLOADS_DIR 未配置")
	}

	if strings.TrimSpace(cfg.Backup.Dir) == "" {
		return fmt.Errorf("BACKUPS_DIR 未配置")
	}

	if len(cfg.CORS.AllowedOrigins) == 0 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS 未配置")
	}

	for _, origin := range cfg.CORS.AllowedOrigins {
		if origin == "*" {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS 不允许使用通配符 *")
		}
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}

	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}

func cleanPath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return filepath.Clean(trimmed)
}
