package handler

import (
	"context"
	"testing"

	"dalanshu/internal/db"
	"dalanshu/internal/model"
	"dalanshu/internal/service"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func testCtx() context.Context { return context.Background() }

func newTestSet(t *testing.T) *db.DBSet {
	t.Helper()
	g, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, _ := g.DB()
	sqlDB.SetMaxOpenConns(1) // 内存库需单连接，避免各连接私有实例
	if err := g.AutoMigrate(
		&model.User{}, &model.Post{}, &model.Comment{},
		&model.Like{}, &model.Collect{}, &model.Help{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return &db.DBSet{Write: g, Read: g}
}

// TestPostCounts 验证 likeCount = base + 真实关系数，且按用户叠加 liked。
func TestPostCounts(t *testing.T) {
	set := newTestSet(t)
	svc := service.NewPostService(set, nil)

	post := model.Post{ID: "p1", Author: "a", Title: "t", Body: "b", BaseLikes: 5, BaseCollects: 2, CreatedAt: 1}
	if err := set.W().Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ToggleLike(testCtx(), 1, "p1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ToggleLike(testCtx(), 2, "p1"); err != nil {
		t.Fatal(err)
	}

	out := svc.ShapeSingle(&post, 1)
	if out.LikeCount != 7 {
		t.Errorf("likeCount got %d want 7", out.LikeCount)
	}
	if !out.Liked {
		t.Error("uid=1 应已点赞")
	}
	outAnon := svc.ShapeSingle(&post, 99)
	if outAnon.Liked {
		t.Error("uid=99 不应有点赞")
	}
	if outAnon.LikeCount != 7 {
		t.Errorf("匿名 likeCount got %d want 7", outAnon.LikeCount)
	}
}

// TestShapePostsBatch 验证列表走批量聚合：计数正确、评论按帖分组，且不带 per-user 状态。
func TestShapePostsBatch(t *testing.T) {
	set := newTestSet(t)
	svc := service.NewPostService(set, nil)

	set.W().Create(&model.Post{ID: "a", Author: "x", Title: "t", Body: "b", BaseLikes: 3, CreatedAt: 3})
	set.W().Create(&model.Post{ID: "b", Author: "y", Title: "t", Body: "b", BaseLikes: 0, CreatedAt: 2})
	set.W().Create(&model.Like{UserID: 1, PostID: "a"})
	set.W().Create(&model.Like{UserID: 2, PostID: "a"})
	set.W().Create(&model.Like{UserID: 1, PostID: "b"})
	set.W().Create(&model.Comment{PostID: "a", Author: "z", Text: "c1", CreatedAt: 1})
	set.W().Create(&model.Comment{PostID: "a", Author: "z", Text: "c2", CreatedAt: 2})

	out, err := svc.ListPosts(testCtx(), 0)
	if err != nil {
		t.Fatal(err)
	}
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

// TestOverlayLikesCopies 验证 per-user 状态叠加互不污染（并发安全的基础）。
func TestOverlayLikesCopies(t *testing.T) {
	set := newTestSet(t)
	set.W().Create(&model.User{ID: 1, Username: "u1"})
	set.W().Create(&model.Post{ID: "a", Author: "x", Title: "t", Body: "b", CreatedAt: 1})
	set.W().Create(&model.Like{UserID: 1, PostID: "a"})
	svc := service.NewPostService(set, nil)

	asUser1, err := svc.ListPosts(testCtx(), 1)
	if err != nil {
		t.Fatal(err)
	}
	asAnon, err := svc.ListPosts(testCtx(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if !asUser1[0].Liked {
		t.Error("uid=1 应已点赞")
	}
	for _, p := range asAnon {
		if p.Liked {
			t.Error("匿名用户不应有点赞状态（共享基础数据被污染）")
		}
	}
}

// TestToggleFlip 验证点赞 toggle 幂等且可来回切换。
func TestToggleFlip(t *testing.T) {
	set := newTestSet(t)
	set.W().Create(&model.Post{ID: "p9", Author: "x", Title: "t", Body: "b", CreatedAt: 1})
	svc := service.NewPostService(set, nil)

	svc.ToggleLike(testCtx(), 7, "p9") // 不存在→插入
	var n int64
	set.W().Model(&model.Like{}).Where("user_id = ? AND post_id = ?", 7, "p9").Count(&n)
	if n != 1 {
		t.Fatalf("插入后 like 数=%d want 1", n)
	}

	svc.ToggleLike(testCtx(), 7, "p9") // 已存在→删除（toggle）
	set.W().Model(&model.Like{}).Where("user_id = ? AND post_id = ?", 7, "p9").Count(&n)
	if n != 0 {
		t.Fatalf("删除后 like 数=%d want 0", n)
	}

	svc.ToggleLike(testCtx(), 7, "p9") // 再次插入
	set.W().Model(&model.Like{}).Where("user_id = ? AND post_id = ?", 7, "p9").Count(&n)
	if n != 1 {
		t.Fatalf("再次插入后 like 数=%d want 1", n)
	}
}
