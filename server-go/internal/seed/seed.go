package seed

import (
	_ "embed"
	"encoding/json"
	"log"
	"time"

	"dalanshu/internal/model"
	"gorm.io/gorm"
)

//go:embed seed.json
var seedData []byte

type seedPost struct {
	ID       string `json:"id"`
	Cat      string `json:"cat"`
	Tall     int    `json:"tall"`
	Author   string `json:"author"`
	Avatar   string `json:"avatar"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	Tags     []string `json:"tags"`
	Likes    int    `json:"likes"`
	Collects int    `json:"collects"`
	Festival int    `json:"festival"`
	Comments []struct {
		Author string `json:"author"`
		Avatar string `json:"avatar"`
		Text   string `json:"text"`
	} `json:"comments"`
}

type seedHelp struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Author string `json:"author"`
	Avatar string `json:"avatar"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	City   string `json:"city"`
	Reward string `json:"reward"`
	Ts     string `json:"ts"`
}

// Seed 首次启动时写入种子；posts 表非空则跳过（保持存量数据）。
func Seed(database *gorm.DB) {
	var data struct {
		Posts []seedPost `json:"posts"`
		Helps []seedHelp `json:"helps"`
	}
	if err := json.Unmarshal(seedData, &data); err != nil {
		log.Fatalf("种子解析失败: %v", err)
	}

	var count int64
	database.Model(&model.Post{}).Count(&count)
	if count > 0 {
		return
	}

	now := time.Now().UnixMilli()
	for i, p := range data.Posts {
		post := model.Post{
			ID:           p.ID,
			Cat:          p.Cat,
			Author:       p.Author,
			Avatar:       p.Avatar,
			Title:        p.Title,
			Body:         p.Body,
			Tags:         model.TagsToJSON(p.Tags),
			Festival:     p.Festival == 1,
			Tall:         p.Tall == 1,
			BaseLikes:    p.Likes,
			BaseCollects: p.Collects,
			CreatedAt:    now - int64(i)*60000,
		}
		if err := database.Create(&post).Error; err != nil {
			log.Fatalf("种子写入帖子失败 %s: %v", p.ID, err)
		}
		for j, c := range p.Comments {
			comment := model.Comment{
				PostID:    p.ID,
				Author:    c.Author,
				Avatar:    c.Avatar,
				Text:      c.Text,
				CreatedAt: post.CreatedAt + int64(j)*1000,
			}
			if err := database.Create(&comment).Error; err != nil {
				log.Fatalf("种子写入评论失败: %v", err)
			}
		}
	}

	for i, hp := range data.Helps {
		help := model.Help{
			ID:        hp.ID,
			Type:      hp.Type,
			Author:    hp.Author,
			Avatar:    hp.Avatar,
			Title:     hp.Title,
			Body:      hp.Body,
			City:      hp.City,
			Reward:    hp.Reward,
			Ts:        hp.Ts,
			CreatedAt: now - int64(i)*60000,
		}
		if err := database.Create(&help).Error; err != nil {
			log.Fatalf("种子写入互助失败 %s: %v", hp.ID, err)
		}
	}

	log.Printf("[seed] 已写入种子：%d 帖 / %d 互助", len(data.Posts), len(data.Helps))
}
