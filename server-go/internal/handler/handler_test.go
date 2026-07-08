package handler

import (
	"testing"

	"dalanshu/internal/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1) // 内存库需单连接，避免各连接私有实例
	if err := db.AutoMigrate(
		&model.User{}, &model.Post{}, &model.Comment{},
		&model.Like{}, &model.Collect{}, &model.Help{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// TestShapePostCounts 验证 likeCount/collectCount = base + 真实关系数，且按用户叠加 liked/collected。
func TestShapePostCounts(t *testing.T) {
	db := newTestDB(t)
	h := New(db, nil, "secret")

	post := model.Post{ID: "p1", Author: "a", Title: "t", Body: "b", BaseLikes: 5, BaseCollects: 2, CreatedAt: 1}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	db.Create(&model.Like{UserID: 1, PostID: "p1"})
	db.Create(&model.Like{UserID: 2, PostID: "p1"})
	db.Create(&model.Collect{UserID: 1, PostID: "p1"})

	out := h.shapePost(&post, 1)
	if out.LikeCount != 7 {
		t.Errorf("likeCount got %d want 7", out.LikeCount)
	}
	if out.CollectCount != 3 {
		t.Errorf("collectCount got %d want 3", out.CollectCount)
	}
	if !out.Liked {
		t.Error("uid=1 应已点赞")
	}
	if !out.Collected {
		t.Error("uid=1 应已收藏")
	}

	outAnon := h.shapePost(&post, 99)
	if outAnon.Liked || outAnon.Collected {
		t.Error("uid=99 不应有点赞/收藏")
	}
	if outAnon.LikeCount != 7 {
		t.Errorf("匿名 likeCount got %d want 7", outAnon.LikeCount)
	}
}

// TestShapePostsBatch 验证列表走批量聚合：计数正确、评论按帖分组，且不带 per-user 状态。
func TestShapePostsBatch(t *testing.T) {
	db := newTestDB(t)
	h := New(db, nil, "secret")

	db.Create(&model.Post{ID: "a", Author: "x", Title: "t", Body: "b", BaseLikes: 3, CreatedAt: 3})
	db.Create(&model.Post{ID: "b", Author: "y", Title: "t", Body: "b", BaseLikes: 0, CreatedAt: 2})
	db.Create(&model.Like{UserID: 1, PostID: "a"})
	db.Create(&model.Like{UserID: 2, PostID: "a"})
	db.Create(&model.Like{UserID: 1, PostID: "b"})
	db.Create(&model.Comment{PostID: "a", Author: "z", Text: "c1", CreatedAt: 1})
	db.Create(&model.Comment{PostID: "a", Author: "z", Text: "c2", CreatedAt: 2})

	var posts []model.Post
	db.Order("created_at DESC").Find(&posts)
	out := h.shapePosts(posts)
	if len(out) != 2 {
		t.Fatalf("len=%d want 2", len(out))
	}
	// 顺序与输入一致（created_at DESC）：a 在前，b 在后
	if out[0].ID != "a" {
		t.Fatalf("out[0].ID=%s want a", out[0].ID)
	}
	if out[0].LikeCount != 5 { // 3 + 2
		t.Errorf("a likeCount=%d want 5", out[0].LikeCount)
	}
	if len(out[0].Comments) != 2 {
		t.Errorf("a comments=%d want 2", len(out[0].Comments))
	}
	if out[1].ID != "b" || out[1].LikeCount != 1 {
		t.Errorf("b likeCount=%d want 1", out[1].LikeCount)
	}
	// 批量组装不应带 per-user 状态（由 overlayLikes 在请求时叠加）
	if out[0].Liked || out[0].Collected {
		t.Error("批量组装不应包含 per-user liked/collected")
	}
}

// TestOverlayLikesCopies 验证 overlayLikes 返回新切片，不修改入参（并发安全）。
func TestOverlayLikesCopies(t *testing.T) {
	db := newTestDB(t)
	db.Create(&model.User{ID: 1, Username: "u1"})
	db.Create(&model.Post{ID: "a", Author: "x", Title: "t", Body: "b", CreatedAt: 1})
	db.Create(&model.Like{UserID: 1, PostID: "a"})
	h := New(db, nil, "secret")

	base := []model.PostOut{{ID: "a"}}
	// 入参应为只读：liked/collected 默认 false
	if base[0].Liked || base[0].Collected {
		t.Fatal("入参不应带状态")
	}
	out := h.overlayLikes(base, 1)
	if !out[0].Liked {
		t.Error("uid=1 应已点赞")
	}
	// 关键：原始 base 必须未被修改（缓存共享对象不可被污染）
	if base[0].Liked {
		t.Error("overlayLikes 不应修改入参切片（存在并发写竞争风险）")
	}
}

// TestFlipToggle 验证点赞 toggle 幂等且可来回切换。
func TestFlipToggle(t *testing.T) {
	db := newTestDB(t)
	h := New(db, nil, "secret")

	h.flip(&model.Like{UserID: 7, PostID: "p9"}, 7, "p9") // 不存在→插入
	var n int64
	db.Model(&model.Like{}).Where("user_id = ? AND post_id = ?", 7, "p9").Count(&n)
	if n != 1 {
		t.Fatalf("插入后 like 数=%d want 1", n)
	}

	h.flip(&model.Like{UserID: 7, PostID: "p9"}, 7, "p9") // 已存在→删除（toggle）
	db.Model(&model.Like{}).Where("user_id = ? AND post_id = ?", 7, "p9").Count(&n)
	if n != 0 {
		t.Fatalf("删除后 like 数=%d want 0", n)
	}

	h.flip(&model.Like{UserID: 7, PostID: "p9"}, 7, "p9") // 再次插入
	db.Model(&model.Like{}).Where("user_id = ? AND post_id = ?", 7, "p9").Count(&n)
	if n != 1 {
		t.Fatalf("再次插入后 like 数=%d want 1", n)
	}
}
