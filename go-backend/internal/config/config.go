package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config 應用配置
type Config struct {
	// 伺服器設定
	ServerPort int

	// 資料庫設定
	DBHost     string
	DBPort     int
	DBName     string
	DBUser     string
	DBPassword string

	// JWT 設定
	JWTSecret     string
	JWTExpireHours int

	// CORS 設定
	CORSOrigins string

	// 日誌設定
	LogDir string
}

// Load 從環境變量載入配置
func Load() *Config {
	return &Config{
		ServerPort:     getEnvInt("SERVER_PORT", 8080),
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnvInt("DB_PORT", 3306),
		DBName:         getEnv("DB_NAME", "gost"),
		DBUser:         getEnv("DB_USER", "root"),
		DBPassword:     getEnv("DB_PASSWORD", ""),
		JWTSecret:      getEnv("JWT_SECRET", ""),
		JWTExpireHours: getEnvInt("JWT_EXPIRE_HOURS", 168), // 7 天
		CORSOrigins:    getEnv("CORS_ORIGINS", "*"),
		LogDir:         getEnv("LOG_DIR", "./logs"),
	}
}

// DSN 產生 MySQL 連線字串
func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
