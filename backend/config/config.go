package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost          string
	DBPort          string
	DBUser          string
	DBPassword      string
	DBName          string
	JWTSecret       string
	ServerPort      string
	DefaultPassword string
	FrontendDist    string
	MaxUploadSize   int64
}

var AppConfig *Config

func LoadConfig() {
	// 优先从项目根目录加载 .env，其次尝试当前目录
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
		log.Println("未找到.env文件,使用默认配置")
	}

	AppConfig = &Config{
		DBHost:          getEnv("DB_HOST", "127.0.0.1"),
		DBPort:          getEnv("DB_PORT", "3307"),
		DBUser:          getEnv("DB_USER", "crane_user"),
		DBPassword:      getEnv("DB_PASSWORD", "zlzk.12345678"),
		DBName:          getEnv("DB_NAME", "crane_system"),
		JWTSecret:       getEnv("JWT_SECRET", "default-secret-key-change-in-production"),
		ServerPort:      getEnv("SERVER_PORT", "8080"),
		DefaultPassword: getEnv("DEFAULT_PASSWORD", "123456"),
		FrontendDist:    getEnv("FRONTEND_DIST", "../frontend/dist"),
		MaxUploadSize:   100 * 1024 * 1024,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
