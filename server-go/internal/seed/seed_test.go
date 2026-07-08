package seed

import (
	"testing"

	"dalanshu/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestSeedLoads 验证内嵌种子可正确解析并写入，且具备幂等性（posts 非空则跳过）。
func TestSeedLoads(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.User{}, &model.Post{}, &model.Comment{}, &model.Help{}); err != nil {
		t.Fatal(err)
	}

	Seed(db)
	var p, h int64
	db.Model(&model.Post{}).Count(&p)
	db.Model(&model.Help{}).Count(&h)
	if p != 21 {
		t.Errorf("种子帖子数=%d want 21", p)
	}
	if h != 5 {
		t.Errorf("种子互助数=%d want 5", h)
	}

	// 幂等：再次 Seed 不应重复写入
	Seed(db)
	db.Model(&model.Post{}).Count(&p)
	if p != 21 {
		t.Errorf("二次 Seed 后帖子数=%d want 21（应幂等）", p)
	}
}
