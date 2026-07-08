package db

import (
	"log"

	"dalanshu/internal/config"
)

// RunMigrate 执行 SQLite↔MySQL 平滑数据迁移（migrate 子命令入口）。
func RunMigrate(cfg *config.Config) {
	from := cfg.MigrateFrom
	if from == "" {
		from = "sqlite"
	}
	to := cfg.MigrateTo
	if to == "" {
		to = "mysql"
	}
	if from == to {
		log.Fatal("迁移源与目标驱动不能相同")
	}
	src, err := Connect(from, dsnFor(cfg, from), "")
	if err != nil {
		log.Fatalf("连接源库(%s)失败: %v", from, err)
	}
	dst, err := Connect(to, dsnFor(cfg, to), "")
	if err != nil {
		log.Fatalf("连接目标库(%s)失败: %v", to, err)
	}
	if err := Migrate(dst); err != nil {
		log.Fatalf("目标库迁移失败: %v", err)
	}
	if err := MigrateData(src, dst); err != nil {
		log.Fatalf("数据迁移失败: %v", err)
	}
	log.Printf("[migrate] 完成：%s -> %s", from, to)
}

func dsnFor(cfg *config.Config, driver string) string {
	if driver == "mysql" {
		if cfg.MySQLDSN == "" {
			log.Fatal("MYSQL_DSN 不能为空")
		}
		return cfg.MySQLDSN
	}
	return cfg.SQLitePath
}
