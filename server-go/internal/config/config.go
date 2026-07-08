package config

import (
	"os"
	"time"
)

// Config 聚合所有外部配置（通过环境变量注入）。
type Config struct {
	Port      string
	JWTSecret string
	// DBDriver 数据库驱动：sqlite（默认，零配置本地运行）或 mysql（生产/高并发）。
	DBDriver string
	// MySQLDSN GORM MySQL DSN，仅 DBDriver=mysql 时需要。
	MySQLDSN string
	// SQLitePath SQLite 文件路径，仅 DBDriver=sqlite 时使用。
	SQLitePath string
	// Redis 可选：未配置则全程直查数据库。
	RedisAddr string
	RedisPass string
	StaticDir string
	GinMode   string
	CacheTTL  time.Duration
	// 迁移子命令：源/目标驱动，默认 sqlite -> mysql。
	MigrateFrom string
	MigrateTo   string
}

func Load() *Config {
	return &Config{
		Port:        getenv("PORT", "8080"),
		JWTSecret:   getenv("JWT_SECRET", "dev-secret-change-me"),
		DBDriver:    getenv("DB_DRIVER", "sqlite"),
		MySQLDSN:    os.Getenv("MYSQL_DSN"),
		SQLitePath:  getenv("SQLITE_PATH", "./data/app.db"),
		RedisAddr:   getenv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPass:   os.Getenv("REDIS_PASS"),
		StaticDir:   os.Getenv("STATIC_DIR"),
		GinMode:     getenv("GIN_MODE", "release"),
		CacheTTL:    30 * time.Second,
		MigrateFrom: getenv("MIGRATE_FROM", "sqlite"),
		MigrateTo:   getenv("MIGRATE_TO", "mysql"),
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
