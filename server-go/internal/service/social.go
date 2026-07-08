package service

import (
	"errors"
	"sort"
	"time"

	"dalanshu/internal/db"
	"dalanshu/internal/model"
)

// 社交域统一哨兵错误，便于 handler 映射 HTTP 状态。
var (
	ErrFollowSelf        = errors.New("follow self")
	ErrMsgEmpty          = errors.New("msg empty")
	ErrMsgSelf           = errors.New("msg self")
	ErrMsgTargetNotFound = errors.New("msg target not found")
)

// SocialService 社交域业务：关注 toggle、用户完整信息、私信会话/线程/发送。
type SocialService struct {
	db *db.DBSet
}

func NewSocialService(d *db.DBSet) *SocialService {
	return &SocialService{db: d}
}

// usernameByID 取用户名（self 校验与 Profile 组装用）。
func (s *SocialService) usernameByID(uid int64) string {
	var u model.User
	if err := s.db.R().First(&u, uid).Error; err != nil {
		return ""
	}
	return u.Username
}

// Follow toggle 关注/取关：不可关注自己；target 不存在由前端保证（对齐 Node 不校验）。
func (s *SocialService) Follow(uid int64, targetName string) error {
	me := s.usernameByID(uid)
	if me == "" {
		return ErrNotFound
	}
	if targetName == me {
		return ErrFollowSelf
	}
	var cnt int64
	s.db.W().Model(&model.Follow{}).Where("follower_id = ? AND target = ?", uid, targetName).Count(&cnt)
	if cnt > 0 {
		return s.db.W().Where("follower_id = ? AND target = ?", uid, targetName).Delete(&model.Follow{}).Error
	}
	return s.db.W().Create(&model.Follow{FollowerID: uid, Target: targetName}).Error
}

// Profile 组装当前用户完整信息（含关注列表/粉丝数/未读私信数）。
func (s *SocialService) Profile(uid int64) (model.UserOut, error) {
	var u model.User
	if err := s.db.R().First(&u, uid).Error; err != nil {
		return model.UserOut{}, ErrNotFound
	}
	var follows []model.Follow
	s.db.R().Where("follower_id = ?", uid).Find(&follows)
	following := make([]string, 0, len(follows))
	for _, f := range follows {
		following = append(following, f.Target)
	}
	var followers, unread int64
	s.db.R().Model(&model.Follow{}).Where("target = ?", u.Username).Count(&followers)
	s.db.R().Model(&model.Message{}).Where("to_name = ? AND read = ?", u.Username, false).Count(&unread)
	return model.UserOut{
		Name:      u.Username,
		Avatar:    u.Avatar,
		Bio:       u.Bio,
		Following: following,
		Followers: int(followers),
		Unread:    int(unread),
	}, nil
}

// AvatarOf 解析用户名头像：优先 users，其次 posts.author，否则默认。
func (s *SocialService) AvatarOf(name string) string {
	var u model.User
	if err := s.db.R().Where("username = ?", name).First(&u).Error; err == nil {
		return u.Avatar
	}
	var p model.Post
	if err := s.db.R().Where("author = ?", name).Order("created_at DESC").First(&p).Error; err == nil {
		return p.Avatar
	}
	return "🙂"
}

// Conversations 返回当前用户会话列表（按对端聚合，最近消息倒序）。
func (s *SocialService) Conversations(uid int64) []model.ConversationOut {
	me := s.usernameByID(uid)
	if me == "" {
		return nil
	}
	var msgs []model.Message
	s.db.R().Where("from_name = ? OR to_name = ?", me, me).Order("created_at DESC").Find(&msgs)

	type agg struct {
		avatar string
		last   string
		ts     int64
		unread int
	}
	convos := map[string]*agg{}
	for _, m := range msgs {
		other := m.FromName
		if m.FromName == me {
			other = m.ToName
		}
		a, ok := convos[other]
		if !ok {
			// 因按 created_at DESC 取，首次遇到即最近消息，写入 last/ts 后不再覆盖。
			a = &agg{avatar: s.AvatarOf(other), last: m.Text, ts: m.CreatedAt}
			convos[other] = a
		}
		if m.ToName == me && !m.Read {
			a.unread++
		}
	}
	out := make([]model.ConversationOut, 0, len(convos))
	for name, a := range convos {
		out = append(out, model.ConversationOut{Name: name, Avatar: a.avatar, Last: a.last, Ts: a.ts, Unread: a.unread})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ts > out[j].Ts })
	return out
}

// Thread 返回与对方的私信线程，并标记对方发来的消息为已读。
func (s *SocialService) Thread(uid int64, otherName string) (model.ThreadOut, error) {
	me := s.usernameByID(uid)
	if me == "" {
		return model.ThreadOut{}, ErrNotFound
	}
	// 标记 other→me 的未读消息为已读。
	s.db.W().Model(&model.Message{}).
		Where("from_name = ? AND to_name = ? AND read = ?", otherName, me, false).
		Update("read", true)

	var msgs []model.Message
	s.db.R().Where("(from_name = ? AND to_name = ?) OR (from_name = ? AND to_name = ?)",
		me, otherName, otherName, me).
		Order("created_at ASC").Find(&msgs)

	out := model.ThreadOut{Name: otherName, Avatar: s.AvatarOf(otherName)}
	out.Messages = make([]model.MessageOut, 0, len(msgs))
	for _, m := range msgs {
		out.Messages = append(out.Messages, model.MessageOut{FromName: m.FromName, Text: m.Text, CreatedAt: m.CreatedAt})
	}
	return out, nil
}

// SendMessage 发送私信：校验 text 非空、非自己、对方为注册用户。
func (s *SocialService) SendMessage(uid int64, to, text string) error {
	me := s.usernameByID(uid)
	if me == "" {
		return ErrNotFound
	}
	if text == "" {
		return ErrMsgEmpty
	}
	if to == me {
		return ErrMsgSelf
	}
	var n int64
	if err := s.db.R().Model(&model.User{}).Where("username = ?", to).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return ErrMsgTargetNotFound
	}
	msg := model.Message{FromName: me, ToName: to, Text: text, Read: false, CreatedAt: time.Now().UnixMilli()}
	return s.db.W().Create(&msg).Error
}
