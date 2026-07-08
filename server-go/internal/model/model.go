package model

import "encoding/json"

// ===== 数据库模型（GORM）=====

type User struct {
	ID           int64  `gorm:"primaryKey;autoIncrement"`
	Username     string `gorm:"uniqueIndex;size:64"`
	PasswordHash string `gorm:"size:128"`
	Avatar       string `gorm:"size:16;default:'😎'"`
	Bio          string `gorm:"size:255;default:''"`
	CreatedAt    int64
}

type Post struct {
	ID           string  `gorm:"primaryKey;size:64"`
	Cat          string  `gorm:"size:32;index"`
	Author       string  `gorm:"size:64"`
	Avatar       string  `gorm:"size:16;default:'😎'"`
	Title        string  `gorm:"size:255"`
	Body         string  `gorm:"type:text"`
	Cover        *string `gorm:"size:255"`
	Tags         string  `gorm:"type:text"`
	Festival     bool
	Tall         bool
	BaseLikes    int `gorm:"default:0"`
	BaseCollects int `gorm:"default:0"`
	CreatedAt    int64 `gorm:"index"`
}

type Comment struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	PostID    string `gorm:"index;size:64"`
	Author    string `gorm:"size:64"`
	Avatar    string `gorm:"size:16;default:'😎'"`
	Text      string `gorm:"type:text"`
	CreatedAt int64
}

// Like / Collect 用复合主键实现「切换（toggle）」语义。
type Like struct {
	UserID int64  `gorm:"primaryKey"`
	PostID string `gorm:"primaryKey;size:64"`
}

type Collect struct {
	UserID int64  `gorm:"primaryKey"`
	PostID string `gorm:"primaryKey;size:64"`
}

type Help struct {
	ID        string `gorm:"primaryKey;size:64"`
	Type      string `gorm:"size:16"`
	Author    string `gorm:"size:64"`
	Avatar    string `gorm:"size:16;default:'😎'"`
	Title     string `gorm:"size:255"`
	Body      string `gorm:"type:text"`
	City      string `gorm:"size:64;default:'同城'"`
	Reward    string `gorm:"size:64"`
	Ts        string `gorm:"size:32;default:'刚刚'"`
	CreatedAt int64
}

// ===== 对外 DTO（与前端契约逐字段一致）=====

type CommentOut struct {
	Author string `json:"author"`
	Avatar string `json:"avatar"`
	Text   string `json:"text"`
}

type PostOut struct {
	ID           string       `json:"id"`
	Cat          string       `json:"cat"`
	Author       string       `json:"author"`
	Avatar       string       `json:"avatar"`
	Title        string       `json:"title"`
	Body         string       `json:"body"`
	Cover        *string      `json:"cover"`
	Tags         []string     `json:"tags"`
	Festival     bool         `json:"festival"`
	Tall         bool         `json:"tall"`
	LikeCount    int          `json:"likeCount"`
	CollectCount int          `json:"collectCount"`
	Liked        bool         `json:"liked"`
	Collected    bool         `json:"collected"`
	Comments     []CommentOut `json:"comments"`
}

type UserOut struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
	Bio    string `json:"bio"`
}

type HelpOut struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Author    string `json:"author"`
	Avatar    string `json:"avatar"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	City      string `json:"city"`
	Reward    string `json:"reward"`
	Ts        string `json:"ts"`
	CreatedAt int64  `json:"created_at"`
}

func ParseTags(s string) []string {
	var t []string
	_ = json.Unmarshal([]byte(s), &t)
	if t == nil {
		return []string{}
	}
	return t
}

func TagsToJSON(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(tags)
	return string(b)
}
