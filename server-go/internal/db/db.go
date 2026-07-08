package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"dalanshu/internal/model"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// Connect 按驱动名建立连接并调优连接池。
//   - driver="sqlite"（默认）：零配置本地运行，复用既有 SQLite 能力。
//   - driver="mysql"：生产 / 高并发，启用较大连接池。
func Connect(driver, dsn string) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch driver {
	case "mysql":
		dialector = mysql.Open(dsn)
	default: // sqlite
		dialector = sqlite.Open(sqliteDSN(dsn))
	}

	database, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := database.DB()
	if err != nil {
		return nil, err
	}
	if driver == "mysql" {
		// 高并发：适度放大连接池并设生命周期，避免长连接堆积与超时。
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetConnMaxLifetime(time.Hour)
		sqlDB.SetConnMaxIdleTime(10 * time.Minute)
	} else {
		// SQLite 写串行化：单连接可彻底规避 "database is locked"；高并发请切 MySQL。
		sqlDB.SetMaxOpenConns(1)
	}
	return database, nil
}

// sqliteDSN 为文件路径补充 WAL / 超时等 PRAGMA，提升并发读吞吐与容错。
func sqliteDSN(path string) string {
	if path == "" {
		path = "./data/app.db"
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0o755)
	}
	return "file:" + path +
		"?_pragma=busy_timeout(10000)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=synchronous(NORMAL)" +
		"&_pragma=foreign_keys(0)"
}

// Migrate 自动建表（幂等）。
func Migrate(database *gorm.DB) error {
	return database.AutoMigrate(
		&model.User{},
		&model.Post{},
		&model.Comment{},
		&model.Like{},
		&model.Collect{},
		&model.Help{},
	)
}

// MigrateData 将 src 的全量数据平滑拷贝到 dst（同构 GORM 模型，可重复执行，目标已非空则跳过）。
// 用于「SQLite 本地库 -> MySQL 生产库」的零停机升级：先在 MySQL 侧 Migrate 建表，再拷贝存量数据。
func MigrateData(src, dst *gorm.DB) error {
	models := []any{
		&model.User{}, &model.Post{}, &model.Comment{},
		&model.Like{}, &model.Collect{}, &model.Help{},
	}
	for _, m := range models {
		if err := copyTable(src, dst, m); err != nil {
			return err
		}
	}
	return nil
}

func copyTable(src, dst *gorm.DB, m any) error {
	// 目标已存在数据则跳过（幂等，避免重复迁移）。
	var n int64
	if err := dst.Model(m).Count(&n).Error; err != nil {
		return fmt.Errorf("统计目标表失败 %T: %w", m, err)
	}
	if n > 0 {
		log.Printf("[migrate] 目标表已非空，跳过: %T", m)
		return nil
	}

	var rows []map[string]any
	if err := src.Model(m).Find(&rows).Error; err != nil {
		return fmt.Errorf("读取源表失败 %T: %w", m, err)
	}
	if len(rows) == 0 {
		return nil
	}
	// 分批写入，避免大表一次性 INSERT 占用过多内存。
	if err := dst.Model(m).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(&rows, 200).Error; err != nil {
		return fmt.Errorf("写入目标表失败 %T: %w", m, err)
	}
	log.Printf("[migrate] 已迁移 %T: %d 行", m, len(rows))
	return nil
}
