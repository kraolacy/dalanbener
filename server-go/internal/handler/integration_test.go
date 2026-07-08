package handler

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"dalanshu/internal/db"
	"dalanshu/internal/model"
	"dalanshu/internal/seed"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	g, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := g.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := g.AutoMigrate(
		&model.User{}, &model.Post{}, &model.Comment{},
		&model.Like{}, &model.Collect{}, &model.Help{},
	); err != nil {
		t.Fatal(err)
	}
	seed.Seed(g)

	set := &db.DBSet{Write: g, Read: g}
	return NewRouter(Deps{DB: set, Cache: nil, Secret: "secret", RateLimit: 0})
}

func do(r *gin.Engine, method, path, token string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHealthEndpoint(t *testing.T) {
	r := setupTestRouter(t)
	w := do(r, "GET", "/api/health", "", nil)
	if w.Code != 200 {
		t.Fatalf("health code %d", w.Code)
	}
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["ok"] != true || body["name"] != "dalanshu" {
		t.Fatalf("health body unexpected: %s", w.Body.String())
	}
}

func TestRegisterLoginFlow(t *testing.T) {
	r := setupTestRouter(t)

	w := do(r, "POST", "/api/register", "", map[string]string{"username": "阿强", "password": "1234", "avatar": "🎣"})
	if w.Code != 200 {
		t.Fatalf("register code %d body %s", w.Code, w.Body.String())
	}
	var reg struct {
		Token string
		User  model.UserOut
	}
	_ = json.Unmarshal(w.Body.Bytes(), &reg)
	if reg.Token == "" || reg.User.Name != "阿强" {
		t.Fatalf("register response unexpected: %s", w.Body.String())
	}
	token := reg.Token

	wMe := do(r, "GET", "/api/me", token, nil)
	if wMe.Code != 200 {
		t.Fatalf("me code %d", wMe.Code)
	}

	longBody := "这是一段足够长的正文，用于验证 tall 字段是否按中文字符数正确计算，必须超过六十个字符才能触发长文标记，否则该断言会失败并暴露长度计算逻辑的问题。"
	wPost := do(r, "POST", "/api/posts", token, map[string]any{
		"title": "测试帖", "body": longBody, "cat": "fishing", "tags": []string{"测试", "钓鱼"},
	})
	if wPost.Code != 200 {
		t.Fatalf("createPost code %d body %s", wPost.Code, wPost.Body.String())
	}
	var p model.PostOut
	_ = json.Unmarshal(wPost.Body.Bytes(), &p)
	if p.Author != "阿强" {
		t.Fatalf("author mismatch: %s", p.Author)
	}
	if !p.Tall {
		t.Fatal("tall 应为 true（长正文）")
	}
	if len(p.Tags) != 2 {
		t.Fatalf("tags 数 %d want 2", len(p.Tags))
	}
	postID := p.ID

	wLike := do(r, "POST", "/api/posts/"+postID+"/like", token, nil)
	if wLike.Code != 200 {
		t.Fatalf("like code %d", wLike.Code)
	}
	var p1 model.PostOut
	_ = json.Unmarshal(wLike.Body.Bytes(), &p1)
	if !p1.Liked || p1.LikeCount != 1 {
		t.Fatalf("点赞后状态异常 liked=%v count=%d", p1.Liked, p1.LikeCount)
	}
	wUnlike := do(r, "POST", "/api/posts/"+postID+"/like", token, nil)
	var p2 model.PostOut
	_ = json.Unmarshal(wUnlike.Body.Bytes(), &p2)
	if p2.Liked || p2.LikeCount != 0 {
		t.Fatalf("再次点赞应取消 liked=%v count=%d", p2.Liked, p2.LikeCount)
	}

	wC := do(r, "POST", "/api/posts/"+postID+"/comments", token, map[string]string{"text": "好帖"})
	if wC.Code != 200 {
		t.Fatalf("comment code %d", wC.Code)
	}
	var p3 model.PostOut
	_ = json.Unmarshal(wC.Body.Bytes(), &p3)
	if len(p3.Comments) != 1 || p3.Comments[0].Text != "好帖" {
		t.Fatalf("评论未写入: %+v", p3.Comments)
	}
}

func TestValidationBranches(t *testing.T) {
	r := setupTestRouter(t)

	if w := do(r, "POST", "/api/register", "", map[string]string{"username": "a", "password": "1234"}); w.Code != 400 {
		t.Fatalf("短用户名应 400，got %d", w.Code)
	}
	if w := do(r, "POST", "/api/register", "", map[string]string{"username": "阿强强", "password": "12"}); w.Code != 400 {
		t.Fatalf("短密码应 400，got %d", w.Code)
	}
	if w := do(r, "POST", "/api/register", "", map[string]string{"username": "用户甲", "password": "1234"}); w.Code != 200 {
		t.Fatalf("注册应 200，got %d", w.Code)
	}
	if w := do(r, "POST", "/api/register", "", map[string]string{"username": "用户甲", "password": "1234"}); w.Code != 409 {
		t.Fatalf("重名应 409，got %d", w.Code)
	}
	if w := do(r, "POST", "/api/login", "", map[string]string{"username": "用户甲", "password": "wrong"}); w.Code != 401 {
		t.Fatalf("错误密码应 401，got %d", w.Code)
	}
	if w := do(r, "POST", "/api/posts", "", map[string]string{"title": "x", "body": "y"}); w.Code != 401 {
		t.Fatalf("未登录发帖应 401，got %d", w.Code)
	}
	if w := do(r, "GET", "/api/me", "", nil); w.Code != 401 {
		t.Fatalf("/me 无 token 应 401，got %d", w.Code)
	}
	if w := do(r, "POST", "/api/posts/nope/like", "faketoken", nil); w.Code != 401 {
		t.Fatalf("伪造 token 应 401，got %d", w.Code)
	}
}

func TestHelpsFlow(t *testing.T) {
	r := setupTestRouter(t)
	w := do(r, "GET", "/api/helps", "", nil)
	if w.Code != 200 {
		t.Fatalf("helps code %d", w.Code)
	}
	var list []model.HelpOut
	_ = json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 5 {
		t.Fatalf("helps 数 %d want 5", len(list))
	}

	reg := do(r, "POST", "/api/register", "", map[string]string{"username": "热心市民", "password": "1234"})
	var regResp struct{ Token string }
	_ = json.Unmarshal(reg.Body.Bytes(), &regResp)
	wH := do(r, "POST", "/api/helps", regResp.Token, map[string]string{"type": "offer", "title": "免费修电脑", "body": "周末义务", "city": "北京"})
	if wH.Code != 200 {
		t.Fatalf("createHelp code %d body %s", wH.Code, wH.Body.String())
	}
	var h model.HelpOut
	_ = json.Unmarshal(wH.Body.Bytes(), &h)
	if h.Type != "offer" || h.Reward != "交个朋友" {
		t.Fatalf("互助字段异常: %+v", h)
	}

	if w := do(r, "POST", "/api/helps", regResp.Token, map[string]string{"type": "offer", "title": "x", "body": ""}); w.Code != 400 {
		t.Fatalf("空 body 应 400，got %d", w.Code)
	}
}

func TestAnonPostsLikedFalse(t *testing.T) {
	r := setupTestRouter(t)
	w := do(r, "GET", "/api/posts", "", nil)
	if w.Code != 200 {
		t.Fatalf("posts code %d", w.Code)
	}
	var list []model.PostOut
	_ = json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) == 0 {
		t.Fatal("帖子流为空")
	}
	for _, p := range list {
		if p.Liked || p.Collected {
			t.Fatalf("匿名用户不应有点赞/收藏状态: %s", p.ID)
		}
	}
}

// TestPostsCursorPagination 验证游标分页：默认返回纯数组，带 limit 返回 {items,next} 且可翻完。
func TestPostsCursorPagination(t *testing.T) {
	r := setupTestRouter(t)

	// 默认（无参数）仍返回纯数组，兼容前端 api.posts()。
	w := do(r, "GET", "/api/posts", "", nil)
	if w.Code != 200 {
		t.Fatalf("posts code %d", w.Code)
	}
	var all []model.PostOut
	_ = json.Unmarshal(w.Body.Bytes(), &all)
	if len(all) != 21 {
		t.Fatalf("默认 posts 数=%d want 21", len(all))
	}

	// 分页：limit=5，逐页翻完，累计应等于 21。
	total := 0
	cursor := ""
	pages := 0
	for {
		path := "/api/posts?limit=5"
		if cursor != "" {
			path += "&cursor=" + cursor
		}
		w := do(r, "GET", path, "", nil)
		if w.Code != 200 {
			t.Fatalf("分页 posts code %d", w.Code)
		}
		var page struct {
			Items []model.PostOut `json:"items"`
			Next  string          `json:"next"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
			t.Fatalf("分页响应解析失败: %v", err)
		}
		if len(page.Items) == 0 {
			t.Fatal("分页返回空页")
		}
		if len(page.Items) > 5 {
			t.Fatalf("单页不应超 limit: %d", len(page.Items))
		}
		total += len(page.Items)
		cursor = page.Next
		pages++
		if cursor == "" {
			break
		}
		if pages > 100 {
			t.Fatal("分页未终止，可能游标失效")
		}
	}
	if total != 21 {
		t.Fatalf("分页累计 posts=%d want 21", total)
	}
}

// TestHelpsCursorPagination 验证互助流游标分页。
func TestHelpsCursorPagination(t *testing.T) {
	r := setupTestRouter(t)
	total := 0
	cursor := ""
	for {
		path := "/api/helps?limit=2"
		if cursor != "" {
			path += "&cursor=" + cursor
		}
		w := do(r, "GET", path, "", nil)
		if w.Code != 200 {
			t.Fatalf("分页 helps code %d", w.Code)
		}
		var page struct {
			Items []model.HelpOut `json:"items"`
			Next  string          `json:"next"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil {
			t.Fatalf("分页响应解析失败: %v", err)
		}
		total += len(page.Items)
		cursor = page.Next
		if cursor == "" {
			break
		}
	}
	if total != 5 {
		t.Fatalf("分页累计 helps=%d want 5", total)
	}
}

// TestRateLimit 验证限流中间件：超出速率后返回 429。
func TestRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	g, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := g.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := g.AutoMigrate(
		&model.User{}, &model.Post{}, &model.Comment{},
		&model.Like{}, &model.Collect{}, &model.Help{},
	); err != nil {
		t.Fatal(err)
	}
	set := &db.DBSet{Write: g, Read: g}
	// 极低速率 + burst=1：首个请求放行，后续立即被限流。
	r := NewRouter(Deps{DB: set, Cache: nil, Secret: "secret", RateLimit: 0.0001})

	if w := do(r, "GET", "/api/health", "", nil); w.Code != 200 {
		t.Fatalf("首个请求应放行，got %d", w.Code)
	}
	got429 := false
	for i := 0; i < 5; i++ {
		if w := do(r, "GET", "/api/health", "", nil); w.Code == 429 {
			got429 = true
			break
		}
	}
	if !got429 {
		t.Fatal("期望触发限流返回 429")
	}
}
