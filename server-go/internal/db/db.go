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

// DBSet 封装读写数据源，支持读写分离：
//   - Write：主库，承载所有写操作与迁移。
//   - Read：从库（可选）；为空时回落到 Write，保证无副本环境仍可运行。
type DBSet struct {
	Write *gorm.DB
	Read  *gorm.DB
}

// R 读数据源：优先从库，未配置则主库。
func (s *DBSet) R() *gorm.DB {
	if s.Read != nil {
		return s.Read
	}
	return s.Write
}

// W 写数据源。
func (s *DBSet) W() *gorm.DB { return s.Write }

// Connect 按驱动建立连接集（可插拔 + 读写分离）。
//   - driver="sqlite"（默认）：零配置本地运行，复用既有 SQLite 能力，忽略 readDSN。
//   - driver="mysql"：生产 / 高并发；readDSN 非空时建立读副本连接。
func Connect(driver, dsn, readDSN string) (*DBSet, error) {
	write, err := open(driver, dsn)
	if err != nil {
		return nil, err
	}
	set := &DBSet{Write: write}
	if readDSN != "" && driver == "mysql" {
		read, err := open(driver, readDSN)
		if err != nil {
			return nil, fmt.Errorf("连接读副本失败: %w", err)
		}
		set.Read = read
		log.Printf("[db] 读写分离已启用：读请求走从库")
	} else {
		set.Read = write
	}
	return set, nil
}

func open(driver, dsn string) (*gorm.DB, error) {
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

// Migrate 自动建表（幂等），作用在写库。
func Migrate(d *DBSet) error {
	return d.W().AutoMigrate(
		&model.User{},
		&model.Post{},
		&model.Comment{},
		&model.Like{},
		&model.Collect{},
		&model.Help{},
		&model.Follow{},
		&model.Message{},
	)
}

// MigrateData 将 src 的全量数据平滑拷贝到 dst（同构 GORM 模型，可重复执行，目标已非空则跳过）。
func MigrateData(src, dst *DBSet) error {
	models := []any{
		&model.User{}, &model.Post{}, &model.Comment{},
		&model.Like{}, &model.Collect{}, &model.Help{},
		&model.Follow{}, &model.Message{},
	}
	for _, m := range models {
		if err := copyTable(src.W(), dst.W(), m); err != nil {
			return err
		}
	}
	return nil
}

func copyTable(src, dst *gorm.DB, m any) error {
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
	if err := dst.Model(m).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(&rows, 200).Error; err != nil {
		return fmt.Errorf("写入目标表失败 %T: %w", m, err)
	}
	log.Printf("[migrate] 已迁移 %T: %d 行", m, len(rows))
	return nil
}
